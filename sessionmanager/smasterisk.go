/*
Real-time Online/Offline Charging System (OCS) for Telecom & ISP environments
Copyright (C) ITsysCOM GmbH

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/
package sessionmanager

import (
	"fmt"
	"net/url"

	"github.com/cgrates/aringo"
	"github.com/cgrates/cgrates/config"
	"github.com/cgrates/cgrates/utils"
	"github.com/cgrates/rpcclient"
)

const (
	CGRAuthAPP = "cgrates_auth"
)

func NewSMAsterisk(cgrCfg *config.CGRConfig, astConnIdx int, smg rpcclient.RpcClientConnection) (*SMAsterisk, error) {
	return &SMAsterisk{cgrCfg: cgrCfg, smg: smg}, nil
}

type SMAsterisk struct {
	cgrCfg     *config.CGRConfig // Separate from smCfg since there can be multiple
	astConnIdx int
	smg        rpcclient.RpcClientConnection
	astConn    *aringo.ARInGO
	astEvChan  chan map[string]interface{}
	astErrChan chan error
}

func (sma *SMAsterisk) connectAsterisk() (err error) {
	connCfg := sma.cgrCfg.SMAsteriskCfg().AsteriskConns[sma.astConnIdx]
	sma.astEvChan = make(chan map[string]interface{})
	sma.astErrChan = make(chan error)
	sma.astConn, err = aringo.NewARInGO(fmt.Sprintf("ws://%s/ari/events?api_key=%s:%s&app=%s", connCfg.Address, connCfg.User, connCfg.Password, CGRAuthAPP), "http://cgrates.org",
		connCfg.User, connCfg.Password, fmt.Sprintf("%s %s", utils.CGRateS, utils.VERSION), sma.astEvChan, sma.astErrChan, connCfg.ConnectAttempts, connCfg.Reconnects)
	if err != nil {
		return err
	}
	return nil
}

// Called to start the service
func (sma *SMAsterisk) ListenAndServe() (err error) {
	if err := sma.connectAsterisk(); err != nil {
		return err
	}
	for {
		select {
		case err = <-sma.astErrChan:
			return err
		case astRawEv := <-sma.astEvChan:
			stasisType := astRawEv["type"].(string)
			if stasisType == "StasisStart" {
				channelData := astRawEv["channel"].(map[string]interface{})
				channelID := channelData["id"].(string)
				if _, err := sma.astConn.Call(aringo.HTTP_POST, fmt.Sprintf("http://%s/ari/applications/%s/subscription?eventSource=channel:%s",
					sma.cgrCfg.SMAsteriskCfg().AsteriskConns[sma.astConnIdx].Address, CGRAuthAPP, channelID), nil); err != nil {
				}
				if _, err := sma.astConn.Call(aringo.HTTP_POST, fmt.Sprintf("http://%s/ari/channels/%s/continue",
					sma.cgrCfg.SMAsteriskCfg().AsteriskConns[sma.astConnIdx].Address, channelID), url.Values{"channelId": {channelID}}); err != nil {
				}
			}
		}
	}
	panic("<SMAsterisk> ListenAndServe out of select")
}

// Called to shutdown the service
func (rls *SMAsterisk) ServiceShutdown() error {
	return nil
}