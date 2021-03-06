// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/mainflux/mainflux/authn"
)

func issueEndpoint(svc authn.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(issueKeyReq)
		if err := req.validate(); err != nil {
			return nil, err
		}

		now := time.Now().UTC()
		newKey := authn.Key{
			IssuedAt: now,
			Type:     req.Type,
		}

		duration := time.Duration(req.Duration * time.Second)
		if duration != 0 {
			exp := now.Add(duration)
			newKey.ExpiresAt = exp
		}

		key, secret, err := svc.Issue(ctx, req.token, newKey)
		if err != nil {
			return nil, err
		}

		res := issueKeyRes{
			ID:       key.ID,
			Value:    secret,
			IssuedAt: key.IssuedAt,
		}
		if !key.ExpiresAt.IsZero() {
			res.ExpiresAt = &key.ExpiresAt
		}
		return res, nil
	}
}

func revokeEndpoint(svc authn.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(keyReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		if err := svc.Revoke(ctx, req.token, req.id); err != nil {
			return nil, err
		}

		return revokeKeyRes{}, nil
	}
}

func retrieveEndpoint(svc authn.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(keyReq)

		if err := req.validate(); err != nil {
			return nil, err
		}

		key, err := svc.Retrieve(ctx, req.token, req.id)

		if err != nil {
			return nil, err
		}
		ret := retrieveKeyRes{
			ID:       key.ID,
			IssuerID: key.IssuerID,
			Subject:  key.Subject,
			Type:     key.Type,
			IssuedAt: key.IssuedAt,
		}
		if !key.ExpiresAt.IsZero() {
			ret.ExpiresAt = &key.ExpiresAt
		}

		return ret, nil
	}
}
