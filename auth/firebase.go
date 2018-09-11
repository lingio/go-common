package auth

import (
	"firebase.google.com/go/auth"
	"firebase.google.com/go"
	"context"
	"google.golang.org/api/option"
	"reflect"
	"errors"
)

type FireBase struct {
	AuthClient *auth.Client
	App        *firebase.App
}

func NewFireBase(ctx context.Context, config *firebase.Config, opts option.ClientOption) (*FireBase, error) {
	fireBaseApp, err := firebase.NewApp(context.Background(), config, opts)
	if err != nil {
		return nil, err
	}
	fireBaseAuthClient, err := fireBaseApp.Auth(ctx)
	if err != nil {
		return nil, err
	}
	return &FireBase{AuthClient: fireBaseAuthClient, App: fireBaseApp}, nil
}

func (f *FireBase) VerifyToken(ctx context.Context, idToken string) (*auth.Token, error) {
	token, err := f.AuthClient.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (f *FireBase) AuthorizeForPrincipal(condition map[string]string, principal interface{}) (bool, error) {
	v := reflect.ValueOf(principal).Elem()
	if v.Kind() != reflect.Struct {
		return false, errors.New("principal must a struct")
	}
	for k, cv := range condition {
		pv := v.FieldByName(k)
		if !pv.IsValid() {
			return false, errors.New("principal don't have key name: " + k)
		}
		if pv.String() != cv {
			return false, nil
		}
	}
	return true, nil
}
