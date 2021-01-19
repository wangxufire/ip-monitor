package main

import (
	"os"
	"reflect"
	"testing"
)

func Test_cnsRecordList(t *testing.T) {
	secretID = os.Getenv("SecretId")
	secretKey = os.Getenv("SecretKey")
	domain = os.Getenv("domain")
	type args struct {
		subDomain string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		wantErr bool
	}{
		{"cnsRecordList test", args{"www"}, map[string]interface{}{"domain": domain}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cnsRecordList(tt.args.subDomain)
			if (err != nil) != tt.wantErr {
				t.Errorf("cnsRecordList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got["data"].(map[string]interface{})["domain"].(map[string]interface{})["name"].(string), tt.want["domain"].(string)) {
				t.Errorf("cnsRecordList() = %v, want %v", got, tt.want)
			}
		})
	}
}
