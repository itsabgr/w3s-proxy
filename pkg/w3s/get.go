package w3s

import (
	"context"
	"fmt"
	"github.com/ipfs/go-cid"
	"net/http"

	w3http "github.com/itsabgr/w3s-proxy/pkg/w3s/http"
)

func (c *client) Get(ctx context.Context, cid cid.Cid) (*w3http.Web3Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/car/%s", c.cfg.endpoint, cid), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.cfg.token))
	req.Header.Add("X-Client", clientName)
	res, err := c.cfg.hc.Do(req)
	return w3http.NewWeb3Response(res, c.bsvc), err
}
