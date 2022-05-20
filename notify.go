package upload

import (
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (u Upload) SendNotify(requuid string) error {

	url, urlError := url.Parse(u.NotifyURL)

	if urlError != nil {
		u.logger.Error("Read caCert error",
			zap.String("requuid", requuid),
			zap.Error(urlError),
		)
		return errors.Wrapf(urlError, "URL Parsing error")
	}

	// https://www.loginradius.com/blog/engineering/http-security-headers/
	// https://www.loginradius.com/blog/engineering/tune-the-go-http-client-for-high-performance/
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 50
	t.MaxConnsPerHost = 50
	t.MaxIdleConnsPerHost = 50
	t.DisableKeepAlives = true
	t.IdleConnTimeout = 30 * time.Second

	if u.MyTlsSetting.InsecureSkipVerify {
		if url.Scheme != "https" {
			u.logger.Error("check Schema insecure",
				zap.String("requuid", requuid),
				zap.Bool("insecure", u.MyTlsSetting.InsecureSkipVerify),
			)
			return errors.New("Parameter 'insecure' makes no sense without Scheme https")
		}
		t.TLSClientConfig.InsecureSkipVerify = true
	}

	if u.MyTlsSetting.CAPath != "" {

		if url.Scheme != "https" {
			u.logger.Error("check Schema capath",
				zap.String("requuid", requuid),
				zap.String("capath", u.MyTlsSetting.CAPath),
			)
			return errors.New("Parameter 'capath' makes no sense without Scheme https")
		}

		caCert, err := ioutil.ReadFile(u.MyTlsSetting.CAPath)
		if err != nil {
			u.logger.Error("Read caCert error",
				zap.String("requuid", requuid),
				zap.Any("capath", u.MyTlsSetting.CAPath),
				zap.Error(err),
			)
			return errors.Wrapf(err, "failed to read capath %q", u.MyTlsSetting.CAPath)
		}
		caCertPool := x509.NewCertPool()
		successful := caCertPool.AppendCertsFromPEM(caCert)
		if !successful {
			u.logger.Error("caCertPool.AppendCertsFromPEM error",
				zap.String("requuid", requuid),
			)
			return errors.New("failed to parse ca certificate as PEM encoded content")
		}
		t.TLSClientConfig.RootCAs = caCertPool
	}

	httpClient := &http.Client{
		Timeout:   5 * time.Second,
		Transport: t,
	}

	// TODO: Handle notify Body
	myRequest, reqerror := http.NewRequestWithContext(u.ctx, u.NotifyMethod, url.String(), nil)

	if reqerror != nil {
		u.logger.Error("httpClient build Request error",
			zap.String("requuid", requuid),
			zap.Any("Request", myRequest),
			zap.Error(reqerror),
		)
		return errors.Wrapf(reqerror, "httpClient build Request error")
	}

	myRequest.Header.Set("User-Agent", "MyUpload-Handler_v"+Version)

	myResp, error := httpClient.Do(myRequest)
	if error != nil {
		u.logger.Error("httpClient Request error",
			zap.String("requuid", requuid),
			zap.Any("Request", myRequest),
			zap.Any("Response", myResp),
			zap.Error(error),
		)
		return errors.Wrapf(error, "httpClient Request error")
	}

	u.logger.Debug("Notify Info",
		zap.Any("Request", myRequest),
		zap.Any("Response", myResp),
	)

	return nil
}
