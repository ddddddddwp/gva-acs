package redact

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestCWMPXMLRedactsOnlyConnectionRequestPassword(t *testing.T) {
	in := []byte(`<soap-env:Envelope xmlns:soap-env="http://schemas.xmlsoap.org/soap/envelope/" xmlns:cwmp="urn:dslforum-org:cwmp-1-2" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><soap-env:Body><cwmp:SetParameterValues><ParameterList><ParameterValueStruct><Name>Device.ManagementServer.ConnectionRequestPassword</Name><Value xsi:type="xsd:string">secret</Value></ParameterValueStruct><ParameterValueStruct><Name>Device.ManagementServer.Password</Name><Value>management-secret</Value></ParameterValueStruct></ParameterList><Password>download-secret</Password></cwmp:SetParameterValues></soap-env:Body></soap-env:Envelope>`)
	original := append([]byte(nil), in...)

	out, err := CWMPXML(in)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out, []byte(">secret<")) {
		t.Fatal("connection request password leaked")
	}
	if !bytes.Contains(out, []byte("******")) {
		t.Fatalf("redaction marker missing: %s", out)
	}
	if !bytes.Contains(out, []byte(">management-secret<")) {
		t.Fatalf("non-matching parameter changed: %s", out)
	}
	if !bytes.Contains(out, []byte(">download-secret<")) {
		t.Fatalf("unrelated Password element changed: %s", out)
	}
	if !bytes.Equal(in, original) {
		t.Fatal("input payload was mutated")
	}
}

func TestCWMPXMLMatchesLocalNamesAcrossNamespaceVariants(t *testing.T) {
	in := []byte(`<Envelope xmlns:p="urn:parameter"><p:ParameterValueStruct><p:Name>Device.ManagementServer.ConnectionRequestPassword</p:Name><p:Value>namespaced-secret</p:Value></p:ParameterValueStruct></Envelope>`)

	out, err := CWMPXML(in)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out, []byte("namespaced-secret")) || !bytes.Contains(out, []byte("******")) {
		t.Fatalf("namespaced parameter was not redacted: %s", out)
	}
}

func TestCWMPXMLMalformedInputFailsClosed(t *testing.T) {
	in := []byte(`<Envelope><ParameterValueStruct><Name>Device.ManagementServer.ConnectionRequestPassword</Name><Value>secret</Envelope>`)

	out, err := CWMPXML(in)
	if err == nil {
		t.Fatal("malformed XML was accepted")
	}
	if len(out) != 0 {
		t.Fatalf("malformed XML returned payload %q", out)
	}
}

func TestCommandJSONRedactsOnlyMatchingParameterAndInternalSentinel(t *testing.T) {
	in := []byte(`{"parameters":[{"name":"Device.ManagementServer.ConnectionRequestPassword","value":"ciphertext-v1:legacy"},{"name":"Device.ManagementServer.Password","value":"other-secret"}],"internal":"__GVA_TR069_CONNECTION_REQUEST_PASSWORD__"}`)
	original := append([]byte(nil), in...)

	out, err := CommandJSON(in)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out, []byte("ciphertext-v1:legacy")) || bytes.Contains(out, []byte("__GVA_TR069_CONNECTION_REQUEST_PASSWORD__")) {
		t.Fatalf("protected value leaked: %s", out)
	}
	var decoded struct {
		Parameters []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"parameters"`
		Internal string `json:"internal"`
	}
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Parameters[0].Value != "******" || decoded.Internal != "******" {
		t.Fatalf("protected values were not redacted: %#v", decoded)
	}
	if decoded.Parameters[1].Value != "other-secret" {
		t.Fatalf("non-matching parameter changed: %#v", decoded.Parameters[1])
	}
	if !bytes.Equal(in, original) {
		t.Fatal("input payload was mutated")
	}
}

func TestCommandJSONMalformedInputFailsClosed(t *testing.T) {
	out, err := CommandJSON([]byte(`{"parameters":[`))
	if err == nil {
		t.Fatal("malformed JSON was accepted")
	}
	if len(out) != 0 {
		t.Fatalf("malformed JSON returned payload %q", out)
	}
}

func TestCommandJSONRedactsRootSentinelAndLegacyCiphertextFields(t *testing.T) {
	root, err := CommandJSON([]byte(`"__GVA_TR069_CONNECTION_REQUEST_PASSWORD__"`))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(root, []byte("__GVA_TR069_CONNECTION_REQUEST_PASSWORD__")) || !bytes.Contains(root, []byte("******")) {
		t.Fatalf("root sentinel leaked: %s", root)
	}

	legacy, err := CommandJSON([]byte(`{"password_ciphertext":"base64-legacy-ciphertext","password":"download-secret"}`))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(legacy, []byte("base64-legacy-ciphertext")) || !bytes.Contains(legacy, []byte("******")) {
		t.Fatalf("legacy ciphertext leaked: %s", legacy)
	}
	if !bytes.Contains(legacy, []byte("download-secret")) {
		t.Fatalf("unrelated password changed: %s", legacy)
	}
}

func TestCommandTextHidesSentinelAndRecognizedLegacyCiphertext(t *testing.T) {
	for _, value := range []string{
		"fault: __GVA_TR069_CONNECTION_REQUEST_PASSWORD__",
		"fault: ciphertext-v1:legacy-value",
	} {
		if got := CommandText(value); got != "******" {
			t.Fatalf("CommandText(%q) = %q, want redaction marker", value, got)
		}
	}
	if got := CommandText("download password failed"); got != "download password failed" {
		t.Fatalf("unrelated text changed: %q", got)
	}
}

func TestConnectionProfileJSONRedactsOnlyRootPassword(t *testing.T) {
	in := []byte(`{"username":"acs-user","password":"profile-secret","unrelated":{"password":"download-secret"}}`)

	out, err := ConnectionProfileJSON(in)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out, []byte("profile-secret")) || !bytes.Contains(out, []byte("******")) {
		t.Fatalf("profile password was not redacted: %s", out)
	}
	if !bytes.Contains(out, []byte("download-secret")) {
		t.Fatalf("unrelated nested password changed: %s", out)
	}
}

func TestConnectionProfileJSONMalformedInputFailsClosed(t *testing.T) {
	out, err := ConnectionProfileJSON([]byte(`{"password":"profile-secret"`))
	if err == nil {
		t.Fatal("malformed profile JSON was accepted")
	}
	if len(out) != 0 {
		t.Fatalf("malformed profile JSON returned payload %q", out)
	}
}
