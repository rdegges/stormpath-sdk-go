package stormpath

import (
	"encoding/base64"
	"errors"
	"net/url"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/nu7hatch/gouuid"
)

//Application represents a Stormpath application object
//
//See: http://docs.stormpath.com/rest/product-guide/#applications
type Application struct {
	resource
	Name                       string                `json:"name,omitempty"`
	Description                string                `json:"description,omitempty"`
	Status                     string                `json:"status,omitempty"`
	CustomData                 *CustomData           `json:"customData,omitempty"`
	Accounts                   *Accounts             `json:"accounts,omitempty"`
	Groups                     *Groups               `json:"groups,omitempty"`
	Tenant                     *Tenant               `json:"tenant,omitempty"`
	PasswordResetTokens        *resource             `json:"passwordResetTokens,omitempty"`
	AccountStoreMappings       *AccountStoreMappings `json:"accountStoreMappings,omitempty"`
	DefaultAccountStoreMapping *AccountStoreMapping  `json:"defaultAccountStoreMapping,omitempty"`
	DefaultGroupStoreMapping   *AccountStoreMapping  `json:"defaultGroupStoreMapping,omitempty"`
}

//Applications represents a paged result or applications
type Applications struct {
	collectionResource
	Items []Application `json:"items"`
}

//IDSiteCallbackResult holds the ID Site callback parsed JWT token information + the acccount if one was given
type IDSiteCallbackResult struct {
	Account *Account
	State   string
	IsNew   bool
	Status  string
}

//NewApplication creates a new application
func NewApplication(name string) *Application {
	return &Application{Name: name}
}

//MakeApplication creates an application resource from an href
func MakeApplication(href string) *Application {
	return &Application{resource: resource{Href: href}}
}

//Load refresh the application resource calling the application href endpoint
func (app *Application) Load() (*Application, error) {
	err := client.get(app.Href, emptyPayload(), app)

	return app, err
}

//Save saves the given application
//
//See: http://docs.stormpath.com/rest/product-guide/#create-an-application-aka-register-an-application-with-stormpath
func (app *Application) Save() error {
	return client.post(app.Href, app, app)
}

//Delete deletes the given applicaiton
//
//See: http://docs.stormpath.com/rest/product-guide/#delete-an-application
func (app *Application) Delete() error {
	return client.delete(app.Href, emptyPayload())
}

//Purge deletes all the account stores before deleting the application
//
//See: http://docs.stormpath.com/rest/product-guide/#delete-an-application
func (app *Application) Purge() error {
	accountStoreMappings, err := app.GetAccountStoreMappings(NewDefaultPageRequest(), NewEmptyFilter())
	if err != nil {
		return err
	}
	for _, m := range accountStoreMappings.Items {
		client.delete(m.AccountStore.Href, emptyPayload())
	}

	return app.Delete()
}

//GetAccountStoreMappings returns all the applications account store mappings
//
//See: http://docs.stormpath.com/rest/product-guide/#application-account-store-mappings
func (app *Application) GetAccountStoreMappings(pageRequest url.Values, filter url.Values) (*AccountStoreMappings, error) {
	accountStoreMappings := &AccountStoreMappings{}

	err := client.get(
		buildAbsoluteURL(app.AccountStoreMappings.Href, requestParams(pageRequest, filter, url.Values{})),
		emptyPayload(),
		accountStoreMappings,
	)

	return accountStoreMappings, err
}

//GetAccounts returns all the accounts of the application
//
//See: http://docs.stormpath.com/rest/product-guide/#application-accounts
func (app *Application) GetAccounts(pageRequest url.Values, filter url.Values) (*Accounts, error) {
	accounts := &Accounts{}

	err := client.get(
		buildAbsoluteURL(app.Accounts.Href, requestParams(pageRequest, filter, url.Values{})),
		emptyPayload(),
		accounts,
	)

	return accounts, err
}

//RegisterAccount registers a new account into the application
//
//See: http://docs.stormpath.com/rest/product-guide/#application-accounts
func (app *Application) RegisterAccount(account *Account) error {
	return client.post(app.Accounts.Href, account, account)
}

//RegisterSocialAccount registers a new account into the application using an external provider Google, Facebook
//
//See: http://docs.stormpath.com/rest/product-guide/#accessing-accounts-with-google-authorization-codes-or-an-access-tokens
func (app *Application) RegisterSocialAccount(socialAccount *SocialAccount) (*Account, error) {
	account := &Account{}

	err := client.post(app.Accounts.Href, socialAccount, account)

	return account, err
}

//AuthenticateAccount authenticates an account against the application
//
//See: http://docs.stormpath.com/rest/product-guide/#authenticate-an-account
func (app *Application) AuthenticateAccount(username string, password string) (*Account, error) {
	accountRef := &accountRef{}
	account := &Account{}

	loginAttemptPayload := make(map[string]string)

	loginAttemptPayload["type"] = "basic"
	loginAttemptPayload["value"] = base64.StdEncoding.EncodeToString([]byte(username + ":" + password))

	err := client.post(buildAbsoluteURL(app.Href, "loginAttempts"), loginAttemptPayload, accountRef)

	if err != nil {
		return nil, err
	}
	account.Href = accountRef.Account.Href

	return account, err
}

//SendPasswordResetEmail sends a password reset email to the given user
//
//See: http://docs.stormpath.com/rest/product-guide/#reset-an-accounts-password
func (app *Application) SendPasswordResetEmail(username string) (*AccountPasswordResetToken, error) {
	passwordResetToken := &AccountPasswordResetToken{}

	passwordResetPayload := make(map[string]string)
	passwordResetPayload["email"] = username

	err := client.post(buildAbsoluteURL(app.Href, "passwordResetTokens"), passwordResetPayload, passwordResetToken)

	return passwordResetToken, err
}

//ValidatePasswordResetToken validates a password reset token
//
//See: http://docs.stormpath.com/rest/product-guide/#reset-an-accounts-password
func (app *Application) ValidatePasswordResetToken(token string) (*AccountPasswordResetToken, error) {
	passwordResetToken := &AccountPasswordResetToken{}

	err := client.get(buildAbsoluteURL(app.Href, "passwordResetTokens", token), emptyPayload(), passwordResetToken)

	return passwordResetToken, err
}

//ResetPassword resets a user password based on the reset token
//
//See: http://docs.stormpath.com/rest/product-guide/#reset-an-accounts-password
func (app *Application) ResetPassword(token string, newPassword string) (*Account, error) {
	accountRef := &accountRef{}
	account := &Account{}

	resetPasswordPayload := make(map[string]string)
	resetPasswordPayload["password"] = newPassword

	err := client.post(buildAbsoluteURL(app.Href, "passwordResetTokens", token), resetPasswordPayload, accountRef)

	if err != nil {
		return nil, err
	}
	account.Href = accountRef.Account.Href

	return account, err
}

//CreateGroup creates a new group in the application
//
//See: http://docs.stormpath.com/rest/product-guide/#application-groups
func (app *Application) CreateGroup(group *Group) error {
	return client.post(app.Groups.Href, group, group)
}

//GetGroups returns all the application groups
//
//See: http://docs.stormpath.com/rest/product-guide/#application-groups
func (app *Application) GetGroups(pageRequest url.Values, filter url.Values) (*Groups, error) {
	groups := &Groups{}

	err := client.get(
		buildAbsoluteURL(app.Groups.Href, requestParams(pageRequest, filter, url.Values{})),
		emptyPayload(),
		groups,
	)

	return groups, err
}

//CreateIDSiteURL creates the IDSite URL for the application
func (app *Application) CreateIDSiteURL(options map[string]string) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)

	nonce, _ := uuid.NewV4()

	if options["path"] == "" {
		options["path"] = "/"
	}

	token.Claims["jti"] = nonce.String()
	token.Claims["iat"] = time.Now().Unix()
	token.Claims["iss"] = client.Credentials.ID
	token.Claims["sub"] = app.Href
	token.Claims["state"] = options["state"]
	token.Claims["path"] = options["path"]
	token.Claims["cb_uri"] = options["callbackURI"]

	tokenString, err := token.SignedString([]byte(client.Credentials.Secret))
	if err != nil {
		return "", err
	}

	p, _ := url.Parse(app.Href)
	ssoURL := p.Scheme + "://" + p.Host + "/sso"

	if options["logout"] == "true" {
		ssoURL = ssoURL + "/logout" + "?jwtRequest=" + tokenString
	} else {
		ssoURL = ssoURL + "?jwtRequest=" + tokenString
	}

	return ssoURL, nil
}

//HandleIDSiteCallback handles the URL from an ID Site callback it parses the JWT token
//validates it and return an IDSiteCallbackResult with the token info + the Account if the sub was given
func (app *Application) HandleIDSiteCallback(URL string) (*IDSiteCallbackResult, error) {
	result := &IDSiteCallbackResult{}

	cbURL, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}

	jwtResponse := cbURL.Query().Get("jwtResponse")

	token, err := jwt.Parse(jwtResponse, func(token *jwt.Token) (interface{}, error) {
		return []byte(client.Credentials.Secret), nil
	})
	if err != nil {
		return nil, err
	}

	if token.Claims["aud"].(string) != client.Credentials.ID {
		return nil, errors.New("ID Site invalid aud")
	}

	if time.Now().Unix() > int64(token.Claims["exp"].(float64)) {
		return nil, errors.New("ID Site JWT has expired")
	}

	if token.Claims["sub"] != nil {
		account, err := MakeAccount(token.Claims["sub"].(string)).Load()
		if err != nil {
			return nil, err
		}
		result.Account = account
	}
	if token.Claims["state"] != nil {
		result.State = token.Claims["state"].(string)
	}
	result.Status = token.Claims["status"].(string)

	return result, nil
}
