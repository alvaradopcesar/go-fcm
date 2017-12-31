package fcm

import (
	"gopkg.in/h2non/gock.v1"
	"net/http"
	"testing"
	"fmt"
)

func TestNewClient(t *testing.T) {
	key := "key"
	client := NewClient(key)

	if client.apiKey != key {
		t.Fatalf("expected apiKey %s", key)
	}

}
func TestClient_SetHTTPClient(t *testing.T) {
	var newHTTPClient *http.Client
	key := "key"
	client := NewClient(key)
	client.SetHTTPClient(newHTTPClient)

	if client.clientHttp != newHTTPClient {
		t.Fatalf("expected clientHttp %v", newHTTPClient)
	}

}

func TestClient_PushSingle(t *testing.T) {
	client := NewClient("key")
	data := map[string]string{
		"body": "Test",
	}

	client.PushSingle("token1", data)

	if client.Message.To == "" {
		t.Error("To is empty")
	}

	if len(client.Message.RegistrationIds) != 0 {
		t.Errorf("expected size 0 got %v", len(client.Message.RegistrationIds))
	}
}

func TestClient_PushMultiple(t *testing.T) {
	client := NewClient("key")
	tokens := []string{"token1", "token2", "token3"}
	data := map[string]string{
		"body": "Test",
	}

	client.PushMultiple(tokens, data)

	if client.Message.To != "" {
		t.Error("To is not empty")
	}

	if len(client.Message.RegistrationIds) != 3 {
		t.Errorf("expected 3, got %d", len(client.Message.RegistrationIds))
	}
}

func TestClient_AppendRegistrationIds(t *testing.T) {
	client := NewClient("key")
	tokens := []string{"token1", "token2", "token3"}
	data := map[string]string{
		"body": "Test",
	}

	client.PushMultiple(tokens, data)

	if len(client.Message.RegistrationIds) != 3 {
		t.Errorf("expected size 3, got %d", len(client.Message.RegistrationIds))
	}

	client.AppendRegistrationIds([]string{"token 4", "token 5"})

	if len(client.Message.RegistrationIds) != 5 {
		t.Errorf("expected size 5, got %d", len(client.Message.RegistrationIds))
	}

}

func TestClient_CleanRegistrationIds(t *testing.T) {
	defer gock.Off()
	tokens := []string{"token 1", "token 2"}
	gock.New(apiIID).
		Get(tokens[0]).
		Reply(http.StatusOK).
		JSON(`{
  				"application":"com.iid.example",
				"authorizedEntity":"123456782354",
			  	"platform":"Android",
			  	"attestStatus":"ROOTED",
			  	"appSigner":"1a2bc3d4e5",
			  	"connectionType":"WIFI",
			  	"connectDate":"2015-05-12",
			  	"rel":{
					"topics":{
				  		"topicname1":{"addDate":"2015-07-30"},
				  		"topicname2":{"addDate":"2015-07-30"},
				  		"topicname3":{"addDate":"2015-07-30"},
				  		"topicname4":{"addDate":"2015-07-30"}
						}
			  		}
	}`)

	gock.New(apiIID).
		Get(tokens[1]).
		Reply(http.StatusOK).
		JSON(`{
  				"error":"InvalidToken",
	}`)


	data := map[string]string{
		"body": "Test",
	}

	// Init client
	client := NewClient("key")
	client.PushMultiple(tokens, data)
	badTokens := client.CleanRegistrationIds()

	if len(badTokens) == 0 {
		t.Errorf("expected 1, got %d", len(badTokens))
	}

	if len(client.Message.RegistrationIds) != 1 {
		t.Errorf("expected 1, got %d", len(client.Message.RegistrationIds))
	}
}

func TestClient_SendErrToManyRegIDs(t *testing.T){
	// Init client
	client := NewClient("key")
	var tokens []string

	for i:= 0; i <= 1000; i++ {
		token := fmt.Sprintf("token %d", i)
		tokens = append(tokens, token)
	}

	data := map[string]string{
		"body": "Test",
	}

	client.PushMultiple(tokens, data)

	_, err := client.Send()

	if err == nil {
		t.Error("expected error is not nil")
	}
	if err.Error() != ErrToManyRegIDs.Error() {
		t.Errorf("expected error too many registrations ids, got %v", err)
	}

}

func TestClient_Send(t *testing.T) {
	// Init client
	client := NewClient("key")

	t.Run("success", func(t *testing.T) {
		defer gock.Off()

		registrationId := "jfey12fugyuy12oijd"

		gock.New(apiFCM).
			Post("").
			Reply(http.StatusOK).
			JSON(`{
				"success": 1,
				"failure":0,
				"results": [{
					"message_id":"1y2t4i224uy2b",
					"registration_id": "jfey12fugyuy12oijd",
					"error": ""
				}]
			}`)

		data := map[string]string{
			"body": "Test",
		}

		client.PushSingle(registrationId, data)

		status, err := client.Send()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if status.StatusCode != http.StatusOK {
			t.Errorf("expected 200 got %d", status.StatusCode)
		}

		if status.Success != 1 {
			t.Errorf("expected 1 got %d", status.Success)
		}

		if status.Failure != 0 {
			t.Errorf("expected 0 got %d", status.Failure)
		}

		if status.Results[0].RegistrationID != registrationId {
			t.Errorf("expected %s, got %s", registrationId, status.Results[0].RegistrationID)
		}

	})

	t.Run("success apply validate data", func(t *testing.T) {
		defer gock.Off()

		registrationId := "jfey12fugyuy12oijd"

		gock.New(apiFCM).
			Post("").
			Reply(http.StatusOK).
			JSON(`{
				"success": 1,
				"failure":0,
				"results": [{
					"message_id":"1y2t4i224uy2b",
					"registration_id": "jfey12fugyuy12oijd",
					"error": ""
				}]
			}`)

		data := map[string]string{
			"body": "Test",
		}

		client.PushSingle(registrationId, data)

		client.Message.TimeToLive = 2419600
		client.Message.Priority = "subnormal"

		status, err := client.Send()
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if client.Message.TimeToLive > 2419200 {
			t.Errorf("expected 2419200, got %d", client.Message.TimeToLive)
		}

		if client.Message.Priority != HighPriority {
			t.Errorf("expected high, got %s", client.Message.Priority)

		}

		if status.StatusCode != http.StatusOK {
			t.Errorf("expected 200 got %d", status.StatusCode)
		}

		if status.Success != 1 {
			t.Errorf("expected 1 got %d", status.Success)
		}

		if status.Failure != 0 {
			t.Errorf("expected 0 got %d", status.Failure)
		}

		if status.Results[0].RegistrationID != registrationId {
			t.Errorf("expected %s, got %s", registrationId, status.Results[0].RegistrationID)
		}

	})

	t.Run("data is empty", func(t *testing.T) {
		defer gock.Off()

		registrationId := "jfey12fugyuy12oijd"
		client.PushSingle(registrationId, nil)

		_, err := client.Send()
		if err == nil {
			t.Error("expected error data is empty")
		}

	})

	t.Run("failure", func(t *testing.T) {
		defer gock.Off()

		gock.New(apiFCM).
			Post("/").
			Reply(400)

		data := map[string]string{
			"body": "Test",
		}

		client.PushSingle("fff", data)

		status, err := client.Send()
		if err == nil {
			t.Error("expected a error")
		}

		if status != nil {
			t.Errorf("expected nil status got %v", status)
		}

	})
}