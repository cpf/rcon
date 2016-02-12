package rcon

// Test assumes you have a local (or docker) running server, listening on 27015, with password "rconpassword"

import "testing"

const hostname string = "localhost"
const port int = 27015
const pw string = "rconpassword"

func TestCreate(t *testing.T) {
	getNewClient()
}

func TestConnect(t *testing.T) {
	c := getNewClient()
	err := c.Connect()
	if nil != err {
		t.Log("Expected no error during connect", err)
		t.Fail()
	}
	defer c.Disconnect()
}

func TestAuthorize(t *testing.T) {
	c := getNewClient()
	err := c.Connect()
	if nil != err {
		t.Log("Expected no error during connect", err)
		t.Fail()
	}
	defer c.Disconnect()

	response, err := c.Authorize()
	if nil != err {
		t.Log("Expected no error during authorize", err)
		t.Fail()
	}
	t.Log("Response: ", response)
}

func TestExecuteStatus(t *testing.T) {
	c := getNewClient()
	err := c.Connect()
	if nil != err {
		t.Log("Expected no error during connect", err)
		t.Fail()
	}
	defer c.Disconnect()

	_, err = c.Authorize()
	if nil != err {
		t.Log("Expected no error during authorize", err)
		t.Fail()
	}

	_, err = c.Execute("status")
	if nil != err {
		t.Log("Expected no error during execute", err)
		t.Fail()
	}
}

func TestWrongPassword(t *testing.T) {
	c := NewClient(hostname, port, "wrong")
	err := c.Connect()
	if nil != err {
		t.Log("Expected no error during connect", err)
		t.Fail()
	}
	defer c.Disconnect()

	_, err = c.Authorize()
	if nil == err {
		t.Fail()
	}
}

func getNewClient() *Client {
	return NewClient(hostname, port, pw)
}
