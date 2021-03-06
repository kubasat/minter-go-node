package api

import (
	"github.com/MinterTeam/minter-go-node/core/state"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/pkg/errors"
	"math/big"
)

type Stake struct {
	Owner    types.Address    `json:"owner"`
	Coin     types.CoinSymbol `json:"coin"`
	Value    string           `json:"value"`
	BipValue string           `json:"bip_value"`
}

type CandidateResponse struct {
	CandidateAddress types.Address `json:"candidate_address"`
	TotalStake       *big.Int      `json:"total_stake"`
	PubKey           types.Pubkey  `json:"pubkey"`
	Commission       uint          `json:"commission"`
	Stakes           []Stake       `json:"stakes,omitempty"`
	CreatedAtBlock   uint          `json:"created_at_block"`
	Status           byte          `json:"status"`
}

func makeResponseCandidate(c state.Candidate, includeStakes bool) CandidateResponse {
	candidate := CandidateResponse{
		CandidateAddress: c.CandidateAddress,
		TotalStake:       c.TotalBipStake,
		PubKey:           c.PubKey,
		Commission:       c.Commission,
		CreatedAtBlock:   c.CreatedAtBlock,
		Status:           c.Status,
	}

	if includeStakes {
		candidate.Stakes = make([]Stake, len(c.Stakes))
		for i, stake := range c.Stakes {
			candidate.Stakes[i] = Stake{
				Owner:    stake.Owner,
				Coin:     stake.Coin,
				Value:    stake.Value.String(),
				BipValue: stake.BipValue.String(),
			}
		}
	}

	return candidate
}

func Candidate(pubkey []byte, height int) (*CandidateResponse, error) {
	cState, err := GetStateForHeight(height)
	if err != nil {
		return nil, err
	}

	candidate := cState.GetStateCandidate(pubkey)
	if candidate == nil {
		return nil, errors.New("Candidate not found")
	}

	response := makeResponseCandidate(*candidate, true)
	return &response, nil
}
