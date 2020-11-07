package notification

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/goharbor/harbor/src/jobservice/job"
	"github.com/goharbor/harbor/src/jobservice/logger"
	"github.com/goharbor/harbor/src/lib/errors"
)

// MSTeamsJob implements the job interface, which send notification to msteams by msteams incoming webhooks.
type MSTeamsJob struct {
	client *http.Client
	logger logger.Interface
}

// MaxFails returns that how many times this job can fail.
func (sj *MSTeamsJob) MaxFails() (result uint) {
	// Default max fails count is 10, and its max retry interval is around 3h
	// Large enough to ensure most situations can notify successfully
	result = 10
	if maxFails, exist := os.LookupEnv(maxFails); exist {
		mf, err := strconv.ParseUint(maxFails, 10, 32)
		if err != nil {
			logger.Warningf("Fetch msteams job maxFails error: %s", err.Error())
			return result
		}
		result = uint(mf)
	}
	return result
}

// MaxCurrency is implementation of same method in Interface.
func (sj *MSTeamsJob) MaxCurrency() uint {
	return 1
}

// ShouldRetry ...
func (sj *MSTeamsJob) ShouldRetry() bool {
	return true
}

// Validate implements the interface in job/Interface
func (sj *MSTeamsJob) Validate(params job.Parameters) error {
	if params == nil {
		// Params are required
		return errors.New("missing parameter of msteams job")
	}

	payload, ok := params["payload"]
	if !ok {
		return errors.Errorf("missing job parameter 'payload'")
	}
	_, ok = payload.(string)
	if !ok {
		return errors.Errorf("malformed job parameter 'payload', expecting string but got %s", reflect.TypeOf(payload).String())
	}

	address, ok := params["address"]
	if !ok {
		return errors.Errorf("missing job parameter 'address'")
	}
	_, ok = address.(string)
	if !ok {
		return errors.Errorf("malformed job parameter 'address', expecting string but got %s", reflect.TypeOf(address).String())
	}
	return nil
}

// Run implements the interface in job/Interface
func (sj *MSTeamsJob) Run(ctx job.Context, params job.Parameters) error {
	if err := sj.init(ctx, params); err != nil {
		return err
	}

	err := sj.execute(params)
	if err != nil {
		sj.logger.Error(err)
	}

	// Wait a second for msteams rate limit, refer to https://api.msteams.com/docs/rate-limits
	time.Sleep(time.Second)
	return err
}

// init msteams job
func (sj *MSTeamsJob) init(ctx job.Context, params map[string]interface{}) error {
	sj.logger = ctx.GetLogger()

	// default use secure transport
	sj.client = httpHelper.clients[secure]
	if v, ok := params["skip_cert_verify"]; ok {
		if skipCertVerify, ok := v.(bool); ok && skipCertVerify {
			// if skip cert verify is true, it means not verify remote cert, use insecure client
			sj.client = httpHelper.clients[insecure]
		}
	}
	return nil
}

// execute msteams job
func (sj *MSTeamsJob) execute(params map[string]interface{}) error {
	payload := params["payload"].(string)
	address := params["address"].(string)

	req, err := http.NewRequest(http.MethodPost, address, bytes.NewReader([]byte(payload)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := sj.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("msteams job(target: %s) response code is %d", address, resp.StatusCode)
	}
	return nil
}
