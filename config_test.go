package main_test

import (
	"testing"
	"time"

	consul "github.com/hashicorp/consul/api"
	"github.com/consulstructure"
	"reflect"
)

func TestKVFromConsul(t *testing.T){

	type Test struct {
		Host string
		Port int
	}

	// Write our test data
	defer testClientWrite(t, testClient(t), map[string]string{
		"test/host": "localhost",
		"test/port": "222222",
	})()

	// write our test struct into consul
	updateCh := make(chan interface{})
	errCh := make(chan error)
	d := &consulstructure.Decoder{
		Target:   &Test{},
		UpdateCh: updateCh,
		ErrCh:    errCh,
	}
	defer d.Close()
	go d.Run()

	var raw interface{}
	select {
	case <-time.After(1 * time.Second):
		t.Fatal("timeout")
	case <-errCh:
		return
	case v := <-updateCh:
		t.Fatalf("got update: %#v", v)
	}

	expected := &Test{Host: "localhost",Port: 222222}
	actual := raw.(*Test)
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func testClient(t *testing.T) *consul.Client {
	client, err := consul.NewClient(consul.DefaultConfig())
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if _, err := client.Status().Leader(); err != nil {
		t.Fatalf("error requesting Consul leader. Is Consul running?\n\n%s", err)
	}

	return client
}

func testClientWrite(t *testing.T, client *consul.Client, data map[string]string) func() {
	for k, v := range data {
		_, err := client.KV().Put(&consul.KVPair{
			Key:   k,
			Value: []byte(v),
		}, nil)
		if err != nil {
			t.Fatalf("error writing to Consul: %s", err)
		}
	}

	return func() {
		for k, _ := range data {
			_, err := client.KV().Delete(k, nil)
			if err != nil {
				t.Fatalf("error deleting from Consul: %s", err)
			}
		}
	}
}