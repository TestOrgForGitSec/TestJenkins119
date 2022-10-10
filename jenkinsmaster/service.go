package jenkinsmaster

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/bndr/gojenkins"
	"github.com/deliveryblueprints/chlog-go/log"
	domain "github.com/deliveryblueprints/chplugin-go/v0.4.0/domainv0_4_0"
	service "github.com/deliveryblueprints/chplugin-go/v0.4.0/servicev0_4_0"
	"github.com/deliveryblueprints/chplugin-service-go/plugin"
	"github.com/google/uuid"
	"strings"
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
	plugin.CHPluginService
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

func (cs *jenkinsMasterService) ExecuteMaster(ctx context.Context, req *service.ExecuteRequest) ([]*domain.MasterResponse, error) {
	ctx = createLogger(req, ctx)
	requestId := ctx.Value("requestId").(string)
	defer log.DestroySubLogger(requestId)

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

	jenkins := gojenkins.CreateJenkins(nil, creds.URL, creds.UserID, creds.Token)

	if _, err := jenkins.Init(ctx); err != nil {
		log.Error(requestId).Err(err).Msg("Unable to initialise Jenkins client")
		return nil, err
	}

	jobs, err := jenkins.GetAllJobs(ctx)
	if err != nil {
		log.Error(requestId).Err(err).Msg("Unable to get Jenkins jobs")
		return nil, err
	}

	var masterResponses []*domain.MasterResponse

	for _, job := range jobs {
		jobType := GetJobClass(job.Raw.Class)

		if jobType == JobClassFolder {
			nestedJobs, err := job.GetInnerJobs(ctx)

			if err != nil {
				log.Error(requestId).Err(err).Msg("Unable to get nested jobs")
				return nil, err
			}

			for _, nestedJob := range nestedJobs {
				nestedJobType := GetJobClass(nestedJob.Raw.Class)
				if nestedJobType == JobClassPipeline {
					masterResponses = append(masterResponses, toMasterResponse(nestedJob.GetDetails()))
				}
			}
		} else if jobType == JobClassPipeline {
			masterResponses = append(masterResponses, toMasterResponse(job.GetDetails()))
		}
	}

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
