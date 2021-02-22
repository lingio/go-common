package common

import (
	"fmt"
	"testing"
)

func TestHttpGet(t *testing.T) {
	type args struct {
		url         string
		bearerToken string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"test1",
			args{
				url:         "https://partner.lingio.com/portal-config",
				bearerToken: "",
			},
		},
		{
			"test1",
			args{
				url:         "https://partner.lingio.com/partners/demo/config",
				bearerToken: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, gotErr := HttpGet(tt.args.url, tt.args.bearerToken)
			if gotErr != nil {
				t.Error("unexpected error")
			}
		})
	}
}

func TestHttpGetNotAuth(t *testing.T) {
	type args struct {
		url         string
		bearerToken string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			"test1",
			args{
				url:         "https://partner.staging.lingio.com/partners/demo/invites",
				bearerToken: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, gotErr := HttpGet(tt.args.url, tt.args.bearerToken)
			if gotErr != nil {
				fmt.Printf("Remote Error: %s", gotErr.Error())
			}
		})
	}
}

/*
func TestHttpPost(t *testing.T) {
	type args struct {
		url         string
		body        interface{}
		bearerToken string
	}
	tests := []struct {
		name  string
		args  args
		want  []byte
		want1 *Error
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := HttpPost(tt.args.url, tt.args.body, tt.args.bearerToken)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HttpPost() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("HttpPost() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestHttpPut(t *testing.T) {
	type args struct {
		url         string
		body        interface{}
		bearerToken string
	}
	tests := []struct {
		name  string
		args  args
		want  []byte
		want1 *Error
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := HttpPut(tt.args.url, tt.args.body, tt.args.bearerToken)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HttpPut() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("HttpPut() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_createBodyBuffer(t *testing.T) {
	type args struct {
		body interface{}
	}
	tests := []struct {
		name  string
		args  args
		want  *bytes.Buffer
		want1 *Error
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := createBodyBuffer(tt.args.body)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createBodyBuffer() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("createBodyBuffer() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_setBearerToken(t *testing.T) {
	type args struct {
		req         *http.Request
		bearerToken string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}

*/
