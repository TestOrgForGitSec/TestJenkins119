package jenkinsmaster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/bndr/gojenkins"
	"github.com/cloudbees-compliance/chlog-go/log"
	domain "github.com/cloudbees-compliance/chplugin-go/v0.4.0/domainv0_4_0"
	service "github.com/cloudbees-compliance/chplugin-go/v0.4.0/servicev0_4_0"
	"github.com/cloudbees-compliance/chplugin-service-go/plugin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

const CredTypePassword = "password"

const JobClassFolder = "Folder"
const JobClassPipeline = "Pipeline"
const JobClassOther = "Other"

var ErrNoUsableCredentials = errors.New("no usable credentials found for account")

type jenkinsCreds struct {
	URL    string `json:"url"`
	UserID string `json:"userId"`
	Token  string `json:"token"`
}

type jenkinsMasterService struct {
	service.CHPluginServiceServer
}

func NewJenkinsMasterService() plugin.CHPluginService {
	return &jenkinsMasterService{}
}

func GetJobClass(Class string) string {
	if strings.HasSuffix(Class, "Folder") {
		return JobClassFolder
	}

	if strings.HasSuffix(Class, "WorkflowJob") {
		return JobClassPipeline
	}

	return JobClassOther

}

func (cs *jenkinsMasterService) GetManifest(context.Context, *service.GetManifestRequest) (*service.GetManifestResponse, error) {
	log.Debug().Msg("Request for manifest")

	return &service.GetManifestResponse{
		Manifest: &domain.Manifest{
			Uuid:    "524bf8d1-65bc-497c-8356-34fd63b96afd",
			Name:    "JenkinsMaster",
			Version: "0.0.1",
			AssetRoles: []*domain.AssetRole{
				{
					AssetType: "PIPELINE",
					Role:      domain.Role_MASTER,
				},
			},
		},
		Error: nil,
	}, nil
}

func (cs *jenkinsMasterService) GetAssetDescriptors(context.Context, *service.GetAssetDescriptorsRequest) (*service.GetAssetDescriptorsResponse, error) {
	var attributeDescriptors []*domain.AssetAttributesDescriptor
	// TODO
	return &service.GetAssetDescriptorsResponse{
		AssetDescriptors: &domain.AssetDescriptors{
			AttributesDescriptors: attributeDescriptors,
		},
	}, nil
}

func (cs *jenkinsMasterService) parseAccount(ac *domain.Account) (*domain.AccountCredential, error) {
	var foundCredentials *domain.AccountCredential
	for _, cred := range ac.AccountCredential {
		if cred.Type == CredTypePassword {
			foundCredentials = cred
			break
		}
	}

	if foundCredentials == nil {
		return nil, ErrNoUsableCredentials
	}

	return foundCredentials, nil
}

func toMasterResponse(jobDetails *gojenkins.JobResponse) *domain.MasterResponse {
	return &domain.MasterResponse{
		Asset: &domain.MasterAsset{
			Type:       "PIPELINE",
			SubType:    "cbci",
			Identifier: jobDetails.URL,
		},
	}
}

func (cs *jenkinsMasterService) ValidateAuthentication(ctx context.Context, req *service.AuthCheckRequest) (*service.AuthCheckResult, error) {
	var result = service.AuthResult_SUCCESS
	ac := req.Account
	credData, err := cs.parseAccount(ac)
	if err != nil {
		result = service.AuthResult_CREDENTIALS_MISSING
	} else {
		var creds jenkinsCreds
		if err := json.Unmarshal([]byte(credData.Credentials), &creds); err != nil {
			log.Error().Err(err).Msg("Unable to unmarshal credentials")
			result = service.AuthResult_CREDENTIALS_MISSING
		} else {
			client := GetHttpClient()
			jenkins := gojenkins.CreateJenkins(&client, creds.URL, creds.UserID, creds.Token)
			if _, err := jenkins.Init(ctx); err != nil {
				log.Error().Err(err).Msgf("Authentication failed")
				result = service.AuthResult_AUTHENTICATION_FAILURE
			}
		}
	}

	return &service.AuthCheckResult{
		Result: &result,
	}, nil
}

func (cs *jenkinsMasterService) getInnerJobs(ctx context.Context, j *gojenkins.Job) ([]*gojenkins.Job, error) {
	var pipelines []*gojenkins.Job
	nestedJobs, err := j.GetInnerJobs(ctx)
	if err != nil {
		return nil, err
	}

	for _, nestedJob := range nestedJobs {
		job := nestedJob
		switch GetJobClass(job.Raw.Class) {
		case JobClassFolder:
			if nextLevel, err := cs.getInnerJobs(ctx, job); err != nil {
				return nil, err
			} else {
				pipelines = append(pipelines, nextLevel...)
			}
		case JobClassPipeline:
			pipelines = append(pipelines, job)
		}
	}

	return pipelines, nil
}

func (cs *jenkinsMasterService) ExecuteMaster(ctx context.Context, req *service.ExecuteRequest, stream service.CHPluginService_MasterServer) ([]*domain.MasterResponse, error) {
	accountFilter := viper.GetString("demo.account.filter")
	if accountFilter == req.Account.Uuid {
		assetFilters := strings.Split(viper.GetString("demo.asset.filter"), ",,,")
		req.AssetIdentifiers = assetFilters
	}

	ctx = createLogger(req, ctx)
	requestId := ctx.Value("requestId").(string)
	defer log.DestroySubLogger(requestId)

	log.Debug(requestId).Msg("Jenkins master execution started")

	ac := req.Account
	credData, err := cs.parseAccount(ac)
	if err != nil {
		return nil, errors.New("failed to parse account details in ExecuteRequest")
	}
	var creds jenkinsCreds
	if err := json.Unmarshal([]byte(credData.Credentials), &creds); err != nil {
		log.Error(requestId).Err(err).Msg("Unable to unmarshal credentials")
		return nil, err
	}
	log.Debug(requestId).Msg("gojenkins.CreateJenkins step start")
	client := GetHttpClient()
	jenkins := gojenkins.CreateJenkins(&client, creds.URL, creds.UserID, creds.Token)
	log.Debug(requestId).Msg("gojenkins.CreateJenkins step end")

	if _, err := jenkins.Init(ctx); err != nil {
		log.Error(requestId).Err(err).Msg("Unable to initialise Jenkins client")
		return nil, err
	}
	log.Debug(requestId).Msg("jenkins.Init passed")

	var jobs []*gojenkins.Job
	if len(req.AssetIdentifiers) == 0 {
		jobs, err = jenkins.GetAllJobs(ctx)
		if err != nil {
			log.Error(requestId).Err(err).Msg("Unable to get Jenkins jobs")
			return nil, err
		}
		log.Debug(requestId).Msgf("jenkins.GetAllJobs passed. %v jobs found", len(jobs))
	} else {

		jobs, err = cs.getSelectedJobs(ctx, jenkins, req.AssetIdentifiers, *log.GetLogger(requestId))
		if err != nil {
			log.Error(requestId).Err(err).Msg("Unable to get Jenkins jobs")
			return nil, err
		}
		log.Debug(requestId).Msgf("jenkins.GetSelectedJobs passed. %v jobs found", len(jobs))
	}

	var masterResponses []*domain.MasterResponse

	for _, job := range jobs {
		switch GetJobClass(job.Raw.Class) {
		case JobClassFolder:
			if nestedJobs, err := cs.getInnerJobs(ctx, job); err != nil {
				log.Error(requestId).Err(err).Msg("Unable to get nested jobs")
				return nil, err
			} else {
				for _, nestedJob := range nestedJobs {
					masterResponses = append(masterResponses, toMasterResponse(nestedJob.GetDetails()))
				}
			}
		case JobClassPipeline:
			masterResponses = append(masterResponses, toMasterResponse(job.GetDetails()))
		}
	}

	log.Debug(requestId).Msgf("Length of response to CE %v", len(masterResponses))
	return masterResponses, nil
}

func createLogger(req *service.ExecuteRequest, ctx context.Context) (contxt context.Context) {

	trackingInfo := make(map[string]string)
	err := json.Unmarshal(req.TrackingInfo, &trackingInfo)
	if err != nil {
		log.Warn().Msg("Unable to unmarshal trackingInfo.")
	}

	requestId := trackingInfo["ch-request-id"]

	if requestId == "" {
		requestId = uuid.New().String()
		trackingInfo["ch-request-id"] = requestId
	}

	ctx = context.WithValue(ctx, "requestId", requestId)
	ctx = context.WithValue(ctx, "trackingInfo", trackingInfo)

	log.CreateSubLogger(requestId, "", trackingInfo)

	return ctx

}
func (cs *jenkinsMasterService) getSelectedJobs(ctx context.Context, jenkins *gojenkins.Jenkins, assetIdentifiers []string, logger zerolog.Logger) ([]*gojenkins.Job, error) {
	var selectedJobs []*gojenkins.Job
	baseURL := jenkins.Server
	_, err := url.Parse(baseURL)
	if err != nil {
		logger.Error().Msgf("Not able to parse base Jenkins URL = %s ", baseURL)
		return nil, err
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	for _, jobUrl := range assetIdentifiers {

		jobId, parentIds, err := extractJobDetails(baseURL, jobUrl, logger)
		if err != nil {
			return nil, err
		}
		if len(jobId) == 0 {
			return nil, errors.New(fmt.Sprintf("Not valid jenkins name found for asset = %s", jobUrl))
		}
		jenkinsJob, err := jenkins.GetJob(ctx, jobId, parentIds...)
		if err != nil {
			return nil, err
		}
		selectedJobs = append(selectedJobs, jenkinsJob)
	}

	return selectedJobs, nil
}

func extractJobDetails(baseURL string, pipeLineURL string, logger zerolog.Logger) (string, []string, error) {
	var parentJobIds []string
	jobId := ""

	if len(pipeLineURL) == 0 || !strings.Contains(pipeLineURL, "/") {
		logger.Error().Msgf("asset Identifier is empty or Invalid %s", pipeLineURL)
		return jobId, parentJobIds, errors.New("asset Identifier is empty or Invalid ")
	}
	_, err := url.ParseRequestURI(pipeLineURL)
	if err != nil {
		logger.Error().Msgf("Not able to parse pipeline URL from asset Id = %s is not valid", pipeLineURL)
		return jobId, parentJobIds, err
	}

	logger.Trace().Msgf("Formed Base URL : %s", baseURL)
	logger.Trace().Msgf("Original Asset URL : %s", pipeLineURL)
	// Remove the base from full url
	result := strings.Split(pipeLineURL, baseURL)
	filteredPath := result[len(result)-1]
	paths := strings.Split(filteredPath, "/")

	for _, splitPath := range paths {
		if len(splitPath) != 0 && splitPath != "job" {
			jobId = splitPath
			parentJobIds = append(parentJobIds, splitPath)
		}
	}
	if len(parentJobIds) > 0 {
		parentJobIds = parentJobIds[:len(parentJobIds)-1]
	}

	logger.Debug().Msgf("Job Id = %v , Parent Ids = %v for asset = %v", jobId, len(parentJobIds), pipeLineURL)
	return jobId, parentJobIds, nil
}

// Empty function definitions required to satisfy the CHPluginServiceServer interface
func (cs *jenkinsMasterService) ExecuteDecorator(context.Context, *service.ExecuteRequest, plugin.AssetFetcher, service.CHPluginService_DecoratorServer) (*service.ExecuteDecoratorResponse, error) {
	return nil, errors.New("Does not  support this role")
}

func (cs *jenkinsMasterService) ExecuteAnalyser(context.Context, *service.ExecuteRequest, plugin.AssetFetcher, service.CHPluginService_AnalyserServer) (*service.ExecuteAnalyserResponse, error) {
	return nil, errors.New("Does not  support this role")
}

func (cs *jenkinsMasterService) ExecuteAggregator(context.Context, *service.ExecuteRequest, plugin.AssetFetcher, service.CHPluginService_AggregatorServer) (*service.ExecuteAggregatorResponse, error) {
	return nil, errors.New("Does not  support this role")
}

func (cs *jenkinsMasterService) ExecuteAssessor(context.Context, *service.ExecuteRequest, plugin.AssetFetcher, service.CHPluginService_AssessorServer) (*service.ExecuteAssessorResponse, error) {
	return nil, errors.New("Does not  support this role")
}
