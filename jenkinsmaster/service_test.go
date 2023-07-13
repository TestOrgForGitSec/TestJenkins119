package jenkinsmaster

import (
	"github.com/cloudbees-compliance/chlog-go/log"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"reflect"
	"testing"
)

func Test_extractJobDetails1(t *testing.T) {
	viper.SetDefault("log.colour", true)
	viper.SetDefault("log.callerinfo", false)
	viper.SetDefault("log.useconsolewriter", true)
	viper.SetDefault("log.unixtime", false)
	viper.SetDefault("log.level", "debug")
	log.Init(viper.GetViper(), map[string]string{"Service": "Jenkins-Master-Unit-Testing"})
	logger := log.GetLogger("Unit-Testing")

	type args struct {
		baseURL     string
		pipeLineURL string
		logger      zerolog.Logger
	}
	tests := []struct {
		name    string
		args    args
		want    string
		want1   []string
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "PipeLine_URL_Normal",
			args: args{
				baseURL:     "https://gauntlet-3.cloudbees.com/compliance-hub",
				pipeLineURL: "https://gauntlet-3.cloudbees.com/compliance-hub/job/BuildJobs/job/compliance-hub-compliance-engine",
				logger:      *logger,
			},
			want:    "compliance-hub-compliance-engine",
			want1:   []string{"BuildJobs"},
			wantErr: false,
		},
		{
			name: "PipeLine_URL_slash_without_sub_job",
			args: args{
				baseURL:     "https://gauntlet-3.cloudbees.com",
				pipeLineURL: "https://gauntlet-3.cloudbees.com/job/compliance-hub-compliance-engine/",
				logger:      *logger,
			},
			want:    "compliance-hub-compliance-engine",
			want1:   []string{},
			wantErr: false,
		},
		{
			name: "PipeLine_URL_slash_with_sub_job",
			args: args{
				baseURL:     "https://gauntlet-3.cloudbees.com/compliance-hub",
				pipeLineURL: "https://gauntlet-3.cloudbees.com/compliance-hub/job/BuildJobs/job/compliance-hub-compliance-engine/",
				logger:      *logger,
			},
			want:    "compliance-hub-compliance-engine",
			want1:   []string{"BuildJobs"},
			wantErr: false,
		},
		{
			name: "PipeLine_URL_Normal_2",
			args: args{
				baseURL:     "https://gauntlet-3.cloudbees.com/compliance-hub/",
				pipeLineURL: "job/BuildJobs/job/compliance-hub-compliance-engine/",
				logger:      *logger,
			},
			want:    "",
			want1:   nil,
			wantErr: true,
		},

		{
			name: "PipeLine_Value_Empty",
			args: args{
				baseURL:     "https://gauntlet-3.cloudbees.com/compliance-hub",
				pipeLineURL: "",
				logger:      *logger,
			},
			want:    "",
			want1:   nil,
			wantErr: true,
		},
		{
			name: "PipeLine_Value_Invalid",
			args: args{
				baseURL:     "https://gauntlet-3.cloudbees.com/compliance-hub",
				pipeLineURL: "wrong-url's",
				logger:      *logger,
			},
			want:    "",
			want1:   nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := extractJobDetails(tt.args.baseURL, tt.args.pipeLineURL, tt.args.logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractJobDetails() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractJobDetails() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("extractJobDetails() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
