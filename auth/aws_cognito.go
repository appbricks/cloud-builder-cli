package auth

import (
	"context"
	"fmt"
	"strconv"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jwt"
	"github.com/mevansam/goutils/logger"

	"github.com/appbricks/cloud-builder/config"

	cbcli_config "github.com/appbricks/cloud-builder-cli/config"
)

type AWSCognitoJWT struct {
	jwkSet    jwk.Set
	jwtToken  jwt.Token
}

func NewAWSCognitoJWT(config config.Config) (*AWSCognitoJWT, error) {

	var (
		err error
	)
	awsJWT := &AWSCognitoJWT{}

	if awsJWT.jwkSet, err = jwk.Fetch(
		context.Background(), 
		fmt.Sprintf(
			"https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", 
			cbcli_config.AWS_COGNITO_REGION, 
			cbcli_config.AWS_COGNITO_USER_POOL_ID,
		),
	); err != nil {
		return nil, err
	}
	if awsJWT.jwtToken, err = jwt.Parse(
		[]byte(config.AuthContext().GetToken().Extra("id_token").(string)),
		jwt.WithKeySet(awsJWT.jwkSet),
	); err != nil {
		return nil, err
	}
	logger.TraceMessage("JWT Token for logged in user is: %# v", awsJWT.jwtToken)

	return awsJWT, nil
}

func (awsJWT *AWSCognitoJWT) UserID() string {
	username, _ := awsJWT.jwtToken.Get("custom:userID")
	return username.(string)
}

func (awsJWT *AWSCognitoJWT) Username() string {
	username, _ := awsJWT.jwtToken.Get("cognito:username")
	return username.(string)
}

func (awsJWT *AWSCognitoJWT) KeyTimestamp() int64 {
	return awsJWT.getTimesampClaim("custom:keyTimestamp")
}

func (awsJWT *AWSCognitoJWT) ConfigTimestamp() int64 {
	return awsJWT.getTimesampClaim("custom:configTimestamp")
}

func (awsJWT *AWSCognitoJWT) getTimesampClaim(claimKey string) int64 {

	var (
		err error
		ok  bool

		value   interface{}
		ts      string
		tsValue int64
	)
	
	if value, _ = awsJWT.jwtToken.Get(claimKey); value == nil {
		return 0
	}	
	if ts, ok = value.(string); !ok {
		logger.ErrorMessage(
			"JWT Token claim %s is not the expected type: %# v", 
			claimKey, value,
		)
		return 0
	}
	if tsValue, err = strconv.ParseInt(ts, 10, 64); err != nil {
		logger.ErrorMessage(
			"Unable to parse JWT Token claim %s with value '%s': %s", 
			claimKey, ts, err.Error(),
		)
		return 0
	}
	return tsValue
}
