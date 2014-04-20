// Copyright 2014 Daniel Pupius

// Package gohubbub provides a PubSubHubbub subscriber client.  It will request
// subscriptions from a hub and handle responses as required by the prootcol.
// Update notifications will be forwarded to the handler function that was
// registered on subscription.
package gohubbub

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Struct for storing information about a subscription.
type subscription struct {
	hub        string
	topic      string
	id         int
	handler    func(string, []byte) // Content-Type, ResponseBody
	lease      time.Duration
	verifiedAt time.Time
}

func (s subscription) String() string {
	return fmt.Sprintf("%s (#%d %s)", s.topic, s.id, s.lease)
}

var nilSubscription = &subscription{}

// A HttpRequester is used to make HTTP requests.  http.Client{} satisfies this
// interface.
type HttpRequester interface {
	Do(req *http.Request) (resp *http.Response, err error)
}

// Client allows you to register a callback for PubSubHubbub subscriptions,
// handlers will be executed when an update is received.
type Client struct {
	// Hostname or IP address that the client will be served from, should be
	// accessible by the hub. e.g. "push.myhost.com"
	self string

	port          int                      // Which port the server will be started on.
	from          string                   // String passed in the "From" header.
	running       bool                     // Whether the server is running.
	subscriptions map[string]*subscription // Map of subscriptions.
	httpRequester HttpRequester            // e.g. http.Client{}.
}

func NewClient(self string, port int, from string) *Client {
	return &Client{
		self,
		port,
		fmt.Sprintf("%s (gohubbub)", from),
		false,
		make(map[string]*subscription),
		&http.Client{}, // TODO: Use client with Timeout transport.
	}
}

// HasSubscription returns true if a subscription exists for the topic.
func (client *Client) HasSubscription(topic string) bool {
	_, ok := client.subscriptions[topic]
	return ok
}

// Discover queries an RSS or Atom feed for the hub which it is publishing to.
func (client *Client) Discover(topic string) (string, error) {
	resp, err := http.Get(topic)
	if err != nil {
		return "", fmt.Errorf("unable to fetch feed, %v", err)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("feed request failed, status code %d", resp.StatusCode)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading feed response, %v", err)
	}

	var f feed
	if xmlError := xml.Unmarshal(body, &f); xmlError != nil {
		return "", fmt.Errorf("unable to parse xml, %v", xmlError)
	}

	links := append(f.Link, f.Channel.Link...)
	for _, link := range links {
		if link.Rel == "hub" {
			return link.Href, nil
		}
	}

	return "", fmt.Errorf("no hub found in feed")
}

// DiscoverAndSubscribe queries an RSS or Atom feed for the hub which it is
// publishing to, then subscribes for updates.
func (client *Client) DiscoverAndSubscribe(topic string, handler func(string, []byte)) error {
	hub, err := client.Discover(topic)
	if err != nil {
		return fmt.Errorf("unable to find hub, %v", err)
	}
	client.Subscribe(hub, topic, handler)
	return nil
}

// Subscribe adds a handler will be called when an update notification is
// received.  If a handler already exists for the given topic it will be
// overridden.
func (client *Client) Subscribe(hub, topic string, handler func(string, []byte)) {
	s := &subscription{
		hub:     hub,
		topic:   topic,
		id:      len(client.subscriptions),
		handler: handler,
	}
	client.subscriptions[topic] = s
	if client.running {
		client.makeSubscriptionRequest(s)
	}
}

// Unsubscribe sends an unsubscribe notification and removes the subscription.
func (client *Client) Unsubscribe(topic string) {
	if s, exists := client.subscriptions[topic]; exists {
		delete(client.subscriptions, topic)
		if client.running {
			client.makeUnsubscribeRequeast(s)
		}
	} else {
		log.Printf("Cannot unsubscribe, %s doesn't exist.", topic)
	}
}

// StartAndServe starts a server using DefaultServeMux, and makes initial
// subscription requests.
func (client *Client) StartAndServe() {
	client.RegisterHandler(http.DefaultServeMux)

	// For default server give other paths a noop endpoint.
	http.HandleFunc("/", client.handleDefaultRequest)

	// Trigger subscription requests async.
	go client.Start()

	log.Printf("Starting HTTP server on port %d", client.port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", client.port), nil))
}

// RegisterHandler binds the client's HandlerFunc to the provided MUX on the
// path /push-callback/
func (client *Client) RegisterHandler(mux *http.ServeMux) {
	mux.HandleFunc("/push-callback/", client.handleCallback)
}

// Start makes the initial subscription requests and marks the client as running.
// Before calling, RegisterHandler should be called with a running server.
func (client *Client) Start() {
	if client.running {
		return
	}

	client.running = true
	client.ensureSubscribed()
}

// String provides a textual representation of the client's current state.
func (client Client) String() string {
	urls := make([]string, len(client.subscriptions))
	i := 0
	for k, _ := range client.subscriptions {
		urls[i] = k
		i++
	}
	return fmt.Sprintf("%d subscription(s): %v", len(client.subscriptions), urls)
}

func (client *Client) ensureSubscribed() {
	for _, s := range client.subscriptions {
		// Try to renew the subscription if the lease expires within an hour.
		oneHourAgo := time.Now().Add(-time.Hour)
		expireTime := s.verifiedAt.Add(s.lease)
		if expireTime.Before(oneHourAgo) {
			client.makeSubscriptionRequest(s)
		}
	}
	time.AfterFunc(time.Minute, client.ensureSubscribed)
}

func (client *Client) makeSubscriptionRequest(s *subscription) {
	callbackUrl := client.formatCallbackURL(s.id)

	log.Println("Subscribing to", s.topic, "waiting for callback on", callbackUrl)

	body := url.Values{}
	body.Set("hub.callback", callbackUrl)
	body.Add("hub.topic", s.topic)
	body.Add("hub.mode", "subscribe")
	// body.Add("hub.lease_seconds", "60")

	req, _ := http.NewRequest("POST", s.hub, bytes.NewBufferString(body.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("From", client.from)

	resp, err := client.httpRequester.Do(req)

	if err != nil {
		log.Printf("Subscription failed, %s, %s", *s, err)

	} else if resp.StatusCode != 202 {
		log.Printf("Subscription failed, %s, status = %s", *s, resp.Status)
	}
}

func (client *Client) makeUnsubscribeRequeast(s *subscription) {
	log.Println("Unsubscribing from", s.topic)

	body := url.Values{}
	body.Set("hub.callback", client.formatCallbackURL(s.id))
	body.Add("hub.topic", s.topic)
	body.Add("hub.mode", "unsubscribe")

	req, _ := http.NewRequest("POST", s.hub, bytes.NewBufferString(body.Encode()))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("From", client.from)

	resp, err := client.httpRequester.Do(req)

	if err != nil {
		log.Printf("Unsubscribe failed, %s, %s", *s, err)

	} else if resp.StatusCode != 202 {
		log.Printf("Unsubscribe failed, %s status = %s", *s, resp.Status)
	}
}

func (client *Client) formatCallbackURL(callback int) string {
	return fmt.Sprintf("http://%s:%d/push-callback/%d", client.self, client.port, callback)
}

func (client *Client) handleDefaultRequest(resp http.ResponseWriter, req *http.Request) {
	resp.Header().Set("Content-Type", "text/plain; charset=utf-8")
	resp.Write([]byte("gohubbub ok"))
	log.Println("Request on", req.URL.Path)
}

func (client *Client) handleCallback(resp http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	requestBody, err := ioutil.ReadAll(req.Body)

	if err != nil {
		log.Printf("Error reading callback request, %s", err)
		return
	}

	params := req.URL.Query()
	topic := params.Get("hub.topic")

	switch params.Get("hub.mode") {
	case "subscribe":
		if s, exists := client.subscriptions[topic]; exists {
			s.verifiedAt = time.Now()
			lease, err := strconv.Atoi(params.Get("hub.lease_seconds"))
			if err == nil {
				s.lease = time.Second * time.Duration(lease)
			}

			log.Printf("Subscription verified for %s, lease is %s", topic, s.lease)
			resp.Write([]byte(params.Get("hub.challenge")))

		} else {
			log.Printf("Unexpected subscription for %s", topic)
			http.Error(resp, "Unexpected subscription", http.StatusBadRequest)
		}

	case "unsubscribe":
		// We optimistically removed the subscription, so only confirm the
		// unsubscribe if no subscription exists for the topic.
		if _, exists := client.subscriptions[topic]; !exists {
			log.Printf("Unsubscribe confirmed for %s", topic)
			resp.Write([]byte(params.Get("hub.challenge")))

		} else {
			log.Printf("Unexpected unsubscribe for %s", topic)
			http.Error(resp, "Unexpected unsubscribe", http.StatusBadRequest)
		}

	case "denied":
		log.Printf("Subscription denied for %s, reason was %s", topic, params.Get("hub.reason"))
		resp.Write([]byte{})
		// TODO: Don't do anything for now, should probably mark the subscription.

	default:
		s, exists := client.subscriptionForPath(req.URL.Path)
		if !exists {
			log.Printf("Callback for unknown subscription: %s", req.URL.String())
			http.Error(resp, "Unknown subscription", http.StatusBadRequest)

		} else {
			log.Printf("Update for %s", s)
			resp.Write([]byte{})

			// Asynchronously notify the subscription handler, shouldn't affect response.
			go s.handler(req.Header.Get("Content-Type"), requestBody)
		}
	}

}

func (client *Client) subscriptionForPath(path string) (*subscription, bool) {
	parts := strings.Split(path, "/")
	if len(parts) != 3 {
		return nilSubscription, false
	}
	id, err := strconv.Atoi(parts[2])
	if err != nil {
		return nilSubscription, false
	}
	for _, s := range client.subscriptions {
		if s.id == id {
			return s, true
		}
	}
	return nilSubscription, false
}

// Protocol cheat sheet:
// ---------------------
//
// SUBSCRIBE
// POST to hub
//
// ContentType: application/x-www-form-urlencoded
// From: gohubbub test app
//
// hub.callback The subscriber's callback URL where notifications should be delivered.
// hub.mode "subscribe" or "unsubscribe"
// hub.topic The topic URL that the subscriber wishes to subscribe to or unsubscribe from.
// hub.lease_seconds Number of seconds for which the subscriber would like to have the subscription active. Hubs MAY choose to respect this value or not, depending on their own policies. This parameter MAY be present for unsubscription requests and MUST be ignored by the hub in that case.
//
// Response should be 202 "Accepted"

// CALLBACK - Denial notification
// Request will have the following query params:
// hub.mode=denied
// hub.topic=[URL that was denied]
// hub.reason=[why it was denied (optional)]

// CALLBACK - Verification
// Request will have the following query params:
// hub.mode=subscribe or unsubscribe
// hub.topic=[URL that was denied]
// hub.challenge=[random string]
// hub.lease_seconds=[how long lease will be held]
//
// Response should be 2xx with hub.challenge in response body.
// 400 to reject

// CALLBACK - Update notification
// Content-Type
// Payload may be a diff
// Link header with rel=hub
// Link header rel=self for topic
//
// Response empty 2xx
