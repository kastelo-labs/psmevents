package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/pmezard/go-difflib/difflib"
)

var (
	version                = ""
	enableUpdateDiff       = true
	enableSessionEvents    = true
	enableSubscriberEvents = true
	enableGroupEvents      = false
	diffContextLines       = 2
)

func init() {
	flag.BoolVar(&enableUpdateDiff, "diff", enableUpdateDiff, "Use diff for object.updated events")
	flag.IntVar(&diffContextLines, "diff-context", diffContextLines, "Number of lines in diff context")
	flag.BoolVar(&enableGroupEvents, "groups", enableGroupEvents, "Subscribe to group events")
	flag.BoolVar(&enableSessionEvents, "sessions", enableSessionEvents, "Subscribe to session events")
	flag.BoolVar(&enableSubscriberEvents, "subscribers", enableSubscriberEvents, "Subscribe to subscriber events")
}

func main() {
	flag.Usage = func() {
		fmt.Println("Usage:")
		fmt.Printf("  %s [options] [PSM JSON-RPC address]\n", os.Args[0])
		fmt.Println("")
		fmt.Println("Options (with their default values):")
		flag.PrintDefaults()
		fmt.Println("")
		fmt.Println("When no address is given, events are read from stdin.")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Printf("  %s < events.json                # parse and clarify events from a capture file\n", os.Args[0])
		fmt.Printf("  %s -diff=false 192.0.2.23:3994  # connect to PSM at 192.0.2.23, subscribe to all events\n", os.Args[0])
	}
	flag.Parse()

	pr := newPeekingDecoder(os.Stdin)
	timestampEvents := false

	if version != "" {
		fmt.Println("psmevents", version)
	}

	if addr := flag.Arg(0); addr != "" {
		if _, _, err := net.SplitHostPort(addr); err != nil {
			addr = net.JoinHostPort(addr, "3994")
		}
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		pr = newPeekingDecoder(conn)

		if enableSessionEvents {
			if err := subscribeSessions(conn, pr); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println("Subscribed to session events")
		}
		if enableSubscriberEvents {
			if err := subscribeSubscribers(conn, pr); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println("Subscribed to subscriber events")
		}
		if enableGroupEvents {
			if err := subscribeGroups(conn, pr); err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Println("Subscribed to group events")
		}

		timestampEvents = true
	}

	i := 0
	for {
		var v interface{}
		if err := pr.Decode(&v); err != nil {
			if err == io.EOF {
				break
			}
			fmt.Println(err)
			return
		}

		if events, ok := v.([]interface{}); ok {
			for _, event := range events {
				event, ok := event.(map[string]interface{})
				if !ok {
					fmt.Println("No an event?")
					continue
				}

				printHeader(i, timestampEvents)
				if !enableUpdateDiff || printEventWithDiff(event, i) != nil {
					printEvent(event, i)
				}
				i++
			}
		} else if event, ok := v.(map[string]interface{}); ok {
			printHeader(i, timestampEvents)
			if !enableUpdateDiff || printEventWithDiff(event, i) != nil {
				printEvent(event, i)
			}
			i++
		}
	}
}

func printHeader(i int, timestamp bool) {
	if timestamp {
		fmt.Printf("*** Event %d at %v ***\n\n", i, time.Now())
	} else {
		fmt.Printf("*** Event %d ***\n\n", i)
	}
}

func printEventWithDiff(event map[string]interface{}, i int) error {
	if method, ok := event["method"].(string); ok && method == "object.updated" {
		params, ok := event["params"].(map[string]interface{})
		if !ok {
			return errors.New("no events params")
		}

		objType, ok := params["type"].(string)
		if !ok {
			return errors.New("no type")
		}
		oldObj, ok := params["oldObject"].(map[string]interface{})
		if !ok {
			return errors.New("no oldObject")
		}
		newObj, ok := params["newObject"].(map[string]interface{})
		if !ok {
			return errors.New("no newObject")
		}

		oldBs, _ := json.MarshalIndent(oldObj, "", "  ")
		newBs, _ := json.MarshalIndent(newObj, "", "  ")
		diff := difflib.UnifiedDiff{
			A:        difflib.SplitLines(string(oldBs)),
			B:        difflib.SplitLines(string(newBs)),
			FromFile: "old " + objType,
			ToFile:   "new " + objType,
			Context:  diffContextLines,
		}

		if text, err := difflib.GetUnifiedDiffString(diff); err != nil {
			return err
		} else {
			fmt.Println(text)
		}

		return nil
	} else {
		return errors.New("not diffable")
	}
}

func printEvent(event map[string]interface{}, i int) {
	if bs, err := json.MarshalIndent(event, "", "  "); err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("%s\n\n", bs)
	}
}

func subscribeSessions(conn net.Conn, pr *peekingDecoder) error {
	_, err := fmt.Fprintln(conn, `{"id":1, "method": "events.setObjectEventFilter", "params":["session", ["created", "updated", "deleted"]]}`)
	if err != nil {
		return err
	}
	return readResult(pr)
}

func subscribeSubscribers(conn net.Conn, pr *peekingDecoder) error {
	_, err := fmt.Fprintln(conn, `{"id":1, "method": "events.setObjectEventFilter", "params":["subscriber", ["created", "updated", "deleted"]]}`)
	if err != nil {
		return err
	}
	return readResult(pr)
}

func subscribeGroups(conn net.Conn, pr *peekingDecoder) error {
	_, err := fmt.Fprintln(conn, `{"id":1, "method": "events.setObjectEventFilter", "params":["group", ["created", "updated", "deleted"]]}`)
	if err != nil {
		return err
	}
	return readResult(pr)
}

func readResult(pr *peekingDecoder) error {
	for {
		b, err := pr.NextByte()
		if err != nil {
			return err
		}
		if b == '{' {
			break
		}

		var ignore interface{}
		if err := pr.Decode(&ignore); err != nil {
			return err
		}
	}

	var v map[string]interface{}
	pr.Decode(&v)

	if err, ok := v["error"]; ok {
		return errors.New(err.(map[string]interface{})["message"].(string))
	}
	return nil
}
