package store

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/golang/glog"

	"github.com/fnproject/cloudevent"
	"github.com/myntra/aggo/pkg/event"
	"github.com/myntra/aggo/pkg/util"
)

type exampleData struct {
	Alpha string `json:"alpha"`
	Beta  int    `json:"beta"`
}

func tptr(t time.Time) *time.Time { return nil }

var testevent = event.Event{
	&cloudevent.CloudEvent{
		EventType:          "myntra.prod.icinga.check_disk",
		EventTypeVersion:   "1.0",
		CloudEventsVersion: "0.1",
		Source:             "/sink",
		EventID:            "42",
		EventTime:          tptr(time.Now()),
		SchemaURL:          "http://www.json.org",
		ContentType:        "application/json",
		Data:               &exampleData{Alpha: "julie", Beta: 42},
		Extensions:         map[string]string{"ext1": "value"},
	},
}

var testRule = event.Rule{
	ID:           "test-rule-id-1",
	HookEndpoint: "http://localhost:3000/testrule",
	HookRetry:    2,
	EventTypes:   []string{"myntra.prod.icinga.check_disk", "myntra.prod.site247.cart_down"},
}

func singleNode(t *testing.T, f func(node *Node)) {

	tmpDir, _ := ioutil.TempDir("", "store_test")
	defer os.RemoveAll(tmpDir)

	// open store
	cfg := &util.Config{
		NodeID:                     "node0",
		RaftBindPort:               6678,
		Dir:                        tmpDir,
		DefaultWaitWindow:          4000, // 3 minutes
		DefaultMaxWaitWindow:       8000, // 6 minutes
		DefaultWaitWindowThreshold: 3800, // 2.5 minutes
		DisablePostHook:            true,
	}

	node, err := NewNode(cfg)
	if err != nil {
		t.Fatal(err)
	}

	glog.Infof("created node sleep 5s")
	// run test
	time.Sleep(time.Second * 5)
	glog.Infof("running test ")
	f(node)

	// close node
	err = node.Shutdown()
	if err != nil {
		t.Fatal(err)
	}

	glog.Info("done test")
}

func TestRuleSingleNode(t *testing.T) {
	singleNode(t, func(node *Node) {

		err := node.AddRule(&testRule)
		if err != nil {
			t.Fatal(err)
		}

		rules := node.GetRules()
		found := false
		for _, rule := range rules {
			if rule.ID == testRule.ID {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("added rule  was not found")
		}

		err = node.RemoveRule(testRule.ID)
		if err != nil {
			t.Fatal(err)
		}

		rules = node.GetRules()
		found = false
		for _, rule := range rules {
			if rule.ID == testRule.ID {
				found = true
				break
			}
		}
		if found {
			t.Fatal("removed rule was found")
		}

	})
}

func TestOrphanEventSingleNode(t *testing.T) {
	singleNode(t, func(node *Node) {
		err := node.Stash(&testevent)
		if err != nil {
			t.Fatal(err)
		}

		var rb *event.RuleBucket
	loop:
		for {
			select {
			case rb = <-node.store.postBucketQueue:
				fmt.Println("rb=>", rb)

			case <-time.After(time.Millisecond * time.Duration(node.store.opt.DefaultWaitWindow+1000)):
				break loop
			}

		}

		if rb != nil {
			t.Fatal("orphan event was stashed")
		}
	})
}

func TestEventSingleNode(t *testing.T) {
	singleNode(t, func(node *Node) {

		err := node.AddRule(&testRule)
		if err != nil {
			t.Fatal(err)
		}

		err = node.Stash(&testevent)
		if err != nil {
			t.Fatal(err)
		}

		var rb *event.RuleBucket
	loop:
		for {
			select {
			case rb = <-node.store.postBucketQueue:
				fmt.Println(rb)

			case <-time.After(time.Millisecond * time.Duration(node.store.opt.DefaultWaitWindow+1000)):
				break loop
			}

		}

		if rb == nil {
			t.Fatal("event was not stashed")
		}
	})
}