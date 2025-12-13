package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"vsC1Y2025V01/src/repository"

	"vsC1Y2025V01/src/auth"
	"vsC1Y2025V01/src/model"
	"vsC1Y2025V01/src/security"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type inMemoryUserRepository struct {
	mu     sync.Mutex
	nextID uint
	users  map[uint]*model.User
}

const testEncryptionKey = "0123456789abcdef0123456789abcdef"

func setEncryptionKeyForTest(t *testing.T) {
	t.Helper()

	previous := os.Getenv("EXCHANGE_CREDENTIALS_KEY")
	encoded := base64.StdEncoding.EncodeToString([]byte(testEncryptionKey))
	if err := os.Setenv("EXCHANGE_CREDENTIALS_KEY", encoded); err != nil {
		t.Fatalf("failed to set encryption key env: %v", err)
	}

	security.ResetEncryptionKeyForTests()
	t.Cleanup(func() {
		if previous == "" {
			_ = os.Unsetenv("EXCHANGE_CREDENTIALS_KEY")
		} else {
			_ = os.Setenv("EXCHANGE_CREDENTIALS_KEY", previous)
		}
		security.ResetEncryptionKeyForTests()
	})
}

func newInMemoryUserRepository() *inMemoryUserRepository {
	return &inMemoryUserRepository{users: make(map[uint]*model.User)}
}

func (r *inMemoryUserRepository) Create(user *model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.nextID++
	now := time.Now()
	clone := *user
	clone.ID = r.nextID
	clone.CreatedAt = now
	clone.UpdatedAt = now
	r.users[clone.ID] = &clone

	user.ID = clone.ID
	user.CreatedAt = clone.CreatedAt
	user.UpdatedAt = clone.UpdatedAt
	return nil
}

func (r *inMemoryUserRepository) FindByUsername(username string) (*model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, user := range r.users {
		if user.Username == username {
			clone := *user
			return &clone, nil
		}
	}

	return nil, repository.ErrUserNotFound
}

func (r *inMemoryUserRepository) FindByID(id uint) (*model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, ok := r.users[id]
	if !ok {
		return nil, repository.ErrUserNotFound
	}

	clone := *user
	return &clone, nil
}

func (r *inMemoryUserRepository) Update(user *model.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	stored, ok := r.users[user.ID]
	if !ok {
		return repository.ErrUserNotFound
	}

	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = time.Now()
	}

	stored.Username = user.Username
	stored.Password = user.Password
	stored.CreatedAt = user.CreatedAt
	stored.UpdatedAt = user.UpdatedAt
	stored.LastLogin = user.LastLogin
	stored.LastSeen = user.LastSeen
	return nil
}

type inMemoryUserExchangeStore struct {
	mu                 sync.Mutex
	nextExchangeID     uint
	nextUserExchangeID uint
	exchanges          map[uint]*model.Exchange
	userExchanges      map[userExchangeKey]*model.UserExchange
}

type userExchangeKey struct {
	UserID     uint
	ExchangeID uint
}

func newInMemoryUserExchangeStore() *inMemoryUserExchangeStore {
	return &inMemoryUserExchangeStore{
		exchanges:     make(map[uint]*model.Exchange),
		userExchanges: make(map[userExchangeKey]*model.UserExchange),
	}
}

func (s *inMemoryUserExchangeStore) CreateExchange(exchange *model.Exchange) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if exchange.ID == 0 {
		s.nextExchangeID++
		exchange.ID = s.nextExchangeID
	}

	clone := *exchange
	s.exchanges[clone.ID] = &clone
	return nil
}

func (s *inMemoryUserExchangeStore) GetExchangeByID(id uint) (*model.Exchange, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	exchange, ok := s.exchanges[id]
	if !ok {
		return nil, repository.ErrExchangeNotFound
	}

	clone := *exchange
	return &clone, nil
}

func (s *inMemoryUserExchangeStore) FindUserExchange(userID, exchangeID uint) (*model.UserExchange, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := userExchangeKey{UserID: userID, ExchangeID: exchangeID}
	ue, ok := s.userExchanges[key]
	if !ok {
		return nil, repository.ErrUserExchangeNotFound
	}

	clone := *ue
	if ex, exists := s.exchanges[exchangeID]; exists {
		exchangeClone := *ex
		clone.Exchange = &exchangeClone
	}

	return &clone, nil
}

func (s *inMemoryUserExchangeStore) SaveUserExchange(ue *model.UserExchange) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := userExchangeKey{UserID: ue.UserID, ExchangeID: ue.ExchangeID}

	if ue.ID == 0 {
		s.nextUserExchangeID++
		ue.ID = s.nextUserExchangeID
		if ue.CreatedAt.IsZero() {
			ue.CreatedAt = time.Now()
		}
	}

	ue.UpdatedAt = time.Now()

	clone := *ue
	if ex, exists := s.exchanges[ue.ExchangeID]; exists {
		exchangeClone := *ex
		clone.Exchange = &exchangeClone
	}

	s.userExchanges[key] = &clone
	return nil
}

func (s *inMemoryUserExchangeStore) ListFormUserExchanges(userID uint) ([]model.UserExchange, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var result []model.UserExchange
	for key, ue := range s.userExchanges {
		if key.UserID != userID || !ue.ShowInForms {
			continue
		}

		clone := *ue
		if ex, exists := s.exchanges[key.ExchangeID]; exists {
			exchangeClone := *ex
			clone.Exchange = &exchangeClone
		}

		result = append(result, clone)
	}

	return result, nil
}

func (s *inMemoryUserExchangeStore) DeleteUserExchange(userID, exchangeID uint) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := userExchangeKey{UserID: userID, ExchangeID: exchangeID}
	if _, ok := s.userExchanges[key]; !ok {
		return false, nil
	}

	delete(s.userExchanges, key)
	return true, nil
}

func TestUserExchangeLifecycle(t *testing.T) {
	setEncryptionKeyForTest(t)

	logger := logrus.NewEntry(logrus.StandardLogger())

	userRepo := newInMemoryUserRepository()
	repository.SetUserRepository(userRepo)
	t.Cleanup(func() {
		repository.SetUserRepository(nil)
	})

	exchangeStore := newInMemoryUserExchangeStore()
	repository.SetUserExchangeStore(exchangeStore)
	t.Cleanup(func() {
		repository.SetUserExchangeStore(nil)
	})

	router := chi.NewRouter()
	router.Post("/auth/register", RegisterHandler(logger))
	router.Post("/auth/login", LoginHandler(logger))
	router.Group(func(r chi.Router) {
		r.Use(auth.RequireAuthMiddleware(logger))
		r.Route("/user-exchanges", func(r chi.Router) {
			r.Post("/", UpsertUserExchangeHandler(logger))
			r.Get("/forms", ListFormUserExchangesHandler(logger))
			r.Delete("/{exchangeID}", DeleteUserExchangeHandler(logger))
		})
	})

	exchange := &model.Exchange{Name: "Binance"}
	if err := exchangeStore.CreateExchange(exchange); err != nil {
		t.Fatalf("failed to seed exchange: %v", err)
	}

	credentials := map[string]string{
		"username": "alice",
		"password": "SuperSecret123",
	}

	body, _ := json.Marshal(credentials)
	registerReq := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	registerRec := httptest.NewRecorder()
	router.ServeHTTP(registerRec, registerReq)
	if registerRec.Code != http.StatusCreated {
		t.Fatalf("expected register 201, got %d", registerRec.Code)
	}

	loginReq := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	if loginRec.Code != http.StatusOK {
		t.Fatalf("expected login 200, got %d", loginRec.Code)
	}

	var tokenCookie *http.Cookie
	for _, cookie := range loginRec.Result().Cookies() {
		if cookie.Name == "token" {
			tokenCookie = cookie
			break
		}
	}
	if tokenCookie == nil {
		t.Fatalf("expected token cookie in login response")
	}

	upsertPayload := map[string]interface{}{
		"exchangeId":    exchange.ID,
		"apiKey":        "initial-api-key",
		"apiSecret":     "initial-api-secret",
		"apiPassphrase": "initial-api-passphrase",
		"showInForms":   true,
	}

	upsertBody, _ := json.Marshal(upsertPayload)
	upsertReq := httptest.NewRequest(http.MethodPost, "/user-exchanges/", bytes.NewReader(upsertBody))
	upsertReq.AddCookie(tokenCookie)
	upsertRec := httptest.NewRecorder()
	router.ServeHTTP(upsertRec, upsertReq)
	if upsertRec.Code != http.StatusOK {
		t.Fatalf("expected upsert 200, got %d", upsertRec.Code)
	}

	var created model.UserExchangeResponse
	if err := json.Unmarshal(upsertRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode created response: %v", err)
	}
	if !created.ShowInForms {
		t.Fatalf("expected showInForms true, got false")
	}
	if !created.HasAPIKey || !created.HasAPISecret || !created.HasAPIPassphrase {
		t.Fatalf("expected all credential flags to be true, got %+v", created)
	}

	user, err := userRepo.FindByUsername("alice")
	if err != nil {
		t.Fatalf("failed to load user: %v", err)
	}

	stored, err := exchangeStore.FindUserExchange(user.ID, exchange.ID)
	if err != nil {
		t.Fatalf("failed to load stored user exchange: %v", err)
	}

	decryptedKey, err := security.DecryptString(stored.APIKeyHash)
	if err != nil {
		t.Fatalf("failed to decrypt stored api key: %v", err)
	}
	if decryptedKey != "initial-api-key" {
		t.Fatalf("expected decrypted api key to match, got %q", decryptedKey)
	}

	decryptedSecret, err := security.DecryptString(stored.APISecretHash)
	if err != nil {
		t.Fatalf("failed to decrypt stored api secret: %v", err)
	}
	if decryptedSecret != "initial-api-secret" {
		t.Fatalf("expected decrypted api secret to match, got %q", decryptedSecret)
	}

	decryptedPassphrase, err := security.DecryptString(stored.APIPassphraseHash)
	if err != nil {
		t.Fatalf("failed to decrypt stored api passphrase: %v", err)
	}
	if decryptedPassphrase != "initial-api-passphrase" {
		t.Fatalf("expected decrypted api passphrase to match, got %q", decryptedPassphrase)
	}

	originalSecretCipher := stored.APISecretHash

	listReq := httptest.NewRequest(http.MethodGet, "/user-exchanges/forms", nil)
	listReq.AddCookie(tokenCookie)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d", listRec.Code)
	}

	var listResp []model.UserExchangeResponse
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to decode list response: %v", err)
	}
	if len(listResp) != 1 {
		t.Fatalf("expected 1 exchange in list, got %d", len(listResp))
	}
	if !listResp[0].HasAPIKey || !listResp[0].HasAPISecret || !listResp[0].HasAPIPassphrase {
		t.Fatalf("expected credential flags to be true in list response, got %+v", listResp[0])
	}

	updatePayload := map[string]interface{}{
		"exchangeId":  exchange.ID,
		"apiSecret":   "updated-api-secret",
		"showInForms": false,
	}
	updateBody, _ := json.Marshal(updatePayload)
	updateReq := httptest.NewRequest(http.MethodPost, "/user-exchanges/", bytes.NewReader(updateBody))
	updateReq.AddCookie(tokenCookie)
	updateRec := httptest.NewRecorder()
	router.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected update 200, got %d", updateRec.Code)
	}

	var updated model.UserExchangeResponse
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("failed to decode update response: %v", err)
	}
	if updated.HasAPISecret == false {
		t.Fatalf("expected update response to indicate secret present")
	}

	storedAfterUpdate, err := exchangeStore.FindUserExchange(user.ID, exchange.ID)
	if err != nil {
		t.Fatalf("failed to load stored user exchange after update: %v", err)
	}
	if storedAfterUpdate.APISecretHash == originalSecretCipher {
		t.Fatalf("expected api secret ciphertext to change after update")
	}
	decryptedUpdatedSecret, err := security.DecryptString(storedAfterUpdate.APISecretHash)
	if err != nil {
		t.Fatalf("failed to decrypt updated api secret: %v", err)
	}
	if decryptedUpdatedSecret != "updated-api-secret" {
		t.Fatalf("expected decrypted updated api secret to match, got %q", decryptedUpdatedSecret)
	}

	listReq = httptest.NewRequest(http.MethodGet, "/user-exchanges/forms", nil)
	listReq.AddCookie(tokenCookie)
	listRec = httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list after update 200, got %d", listRec.Code)
	}

	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to decode list response after update: %v", err)
	}
	if len(listResp) != 0 {
		t.Fatalf("expected 0 exchanges after hiding, got %d", len(listResp))
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/user-exchanges/"+strconv.Itoa(int(exchange.ID)), nil)
	deleteReq.AddCookie(tokenCookie)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected delete 204, got %d", deleteRec.Code)
	}

	listReq = httptest.NewRequest(http.MethodGet, "/user-exchanges/forms", nil)
	listReq.AddCookie(tokenCookie)
	listRec = httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected list after delete 200, got %d", listRec.Code)
	}

	if err := json.Unmarshal(listRec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to decode list response after delete: %v", err)
	}
	if len(listResp) != 0 {
		t.Fatalf("expected no exchanges after delete, got %d", len(listResp))
	}
}

type stubMexcConnector struct {
	err error
}

func (s *stubMexcConnector) TestConnection() error {
	return s.err
}

func TestTestMexcConnectionHandler_Success(t *testing.T) {
	setEncryptionKeyForTest(t)

	logger := logrus.NewEntry(logrus.StandardLogger())

	exchangeStore := newInMemoryUserExchangeStore()
	repository.SetUserExchangeStore(exchangeStore)
	t.Cleanup(func() { repository.SetUserExchangeStore(nil) })

	mexcExchange := &model.Exchange{Name: "MEXC"}
	if err := exchangeStore.CreateExchange(mexcExchange); err != nil {
		t.Fatalf("failed to seed mexc exchange: %v", err)
	}

	user := &model.User{ID: 42, Username: "bob"}

	encryptedKey, err := security.EncryptString("mexc-key")
	if err != nil {
		t.Fatalf("failed to encrypt api key: %v", err)
	}

	encryptedSecret, err := security.EncryptString("mexc-secret")
	if err != nil {
		t.Fatalf("failed to encrypt api secret: %v", err)
	}

	ue := &model.UserExchange{
		UserID:        user.ID,
		ExchangeID:    mexcExchange.ID,
		APIKeyHash:    encryptedKey,
		APISecretHash: encryptedSecret,
	}
	if err := exchangeStore.SaveUserExchange(ue); err != nil {
		t.Fatalf("failed to seed user exchange: %v", err)
	}

	origFactory := mexcConnectorFactory
	defer func() { mexcConnectorFactory = origFactory }()

	var receivedKey, receivedSecret string
	mexcConnectorFactory = func(apiKey, apiSecret string) mexcConnector {
		receivedKey = apiKey
		receivedSecret = apiSecret
		return &stubMexcConnector{}
	}

	req := httptest.NewRequest(http.MethodPost, "/user-exchanges/"+strconv.Itoa(int(mexcExchange.ID))+"/test", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("exchangeID", strconv.Itoa(int(mexcExchange.ID)))
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)
	ctx = context.WithValue(ctx, auth.UserKey, user)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler := TestMexcConnectionHandler(logger)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}

	if receivedKey != "mexc-key" {
		t.Fatalf("expected connector to receive API key, got %q", receivedKey)
	}
	if receivedSecret != "mexc-secret" {
		t.Fatalf("expected connector to receive API secret, got %q", receivedSecret)
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", resp["status"])
	}
}

func TestTestMexcConnectionHandler_Failure(t *testing.T) {
	setEncryptionKeyForTest(t)

	logger := logrus.NewEntry(logrus.StandardLogger())

	exchangeStore := newInMemoryUserExchangeStore()
	repository.SetUserExchangeStore(exchangeStore)
	t.Cleanup(func() { repository.SetUserExchangeStore(nil) })

	mexcExchange := &model.Exchange{Name: "MEXC"}
	if err := exchangeStore.CreateExchange(mexcExchange); err != nil {
		t.Fatalf("failed to seed mexc exchange: %v", err)
	}

	user := &model.User{ID: 7, Username: "alice"}
	encryptedKey, err := security.EncryptString("mexc-key")
	if err != nil {
		t.Fatalf("failed to encrypt api key: %v", err)
	}
	encryptedSecret, err := security.EncryptString("mexc-secret")
	if err != nil {
		t.Fatalf("failed to encrypt api secret: %v", err)
	}

	ue := &model.UserExchange{
		UserID:        user.ID,
		ExchangeID:    mexcExchange.ID,
		APIKeyHash:    encryptedKey,
		APISecretHash: encryptedSecret,
	}
	if err := exchangeStore.SaveUserExchange(ue); err != nil {
		t.Fatalf("failed to seed user exchange: %v", err)
	}

	origFactory := mexcConnectorFactory
	defer func() { mexcConnectorFactory = origFactory }()

	stubErr := errors.New("connection failed")
	mexcConnectorFactory = func(apiKey, apiSecret string) mexcConnector {
		return &stubMexcConnector{err: stubErr}
	}

	req := httptest.NewRequest(http.MethodPost, "/user-exchanges/"+strconv.Itoa(int(mexcExchange.ID))+"/test", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("exchangeID", strconv.Itoa(int(mexcExchange.ID)))
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)
	ctx = context.WithValue(ctx, auth.UserKey, user)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler := TestMexcConnectionHandler(logger)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "Failed to connect to MEXC") {
		t.Fatalf("expected failure message, got %q", rr.Body.String())
	}
}

func TestTestMexcConnectionHandler_NotMexc(t *testing.T) {
	setEncryptionKeyForTest(t)

	logger := logrus.NewEntry(logrus.StandardLogger())

	exchangeStore := newInMemoryUserExchangeStore()
	repository.SetUserExchangeStore(exchangeStore)
	t.Cleanup(func() { repository.SetUserExchangeStore(nil) })

	otherExchange := &model.Exchange{Name: "Binance"}
	if err := exchangeStore.CreateExchange(otherExchange); err != nil {
		t.Fatalf("failed to seed exchange: %v", err)
	}

	user := &model.User{ID: 11}
	encryptedKey, err := security.EncryptString("binance-key")
	if err != nil {
		t.Fatalf("failed to encrypt api key: %v", err)
	}
	encryptedSecret, err := security.EncryptString("binance-secret")
	if err != nil {
		t.Fatalf("failed to encrypt api secret: %v", err)
	}

	ue := &model.UserExchange{
		UserID:        user.ID,
		ExchangeID:    otherExchange.ID,
		APIKeyHash:    encryptedKey,
		APISecretHash: encryptedSecret,
	}
	if err := exchangeStore.SaveUserExchange(ue); err != nil {
		t.Fatalf("failed to seed user exchange: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/user-exchanges/"+strconv.Itoa(int(otherExchange.ID))+"/test", nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("exchangeID", strconv.Itoa(int(otherExchange.ID)))
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)
	ctx = context.WithValue(ctx, auth.UserKey, user)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	handler := TestMexcConnectionHandler(logger)
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "exchange is not MEXC") {
		t.Fatalf("expected not MEXC message, got %q", rr.Body.String())
	}
}

type mockMexcConnector struct {
	err    error
	called int
}

func (m *mockMexcConnector) TestConnection() error {
	m.called++
	return m.err
}

func TestTestMexcConnectionHandlerSuccess(t *testing.T) {
	setEncryptionKeyForTest(t)

	logger := logrus.NewEntry(logrus.StandardLogger())

	exchangeStore := newInMemoryUserExchangeStore()
	repository.SetUserExchangeStore(exchangeStore)
	t.Cleanup(func() {
		repository.SetUserExchangeStore(nil)
	})

	exchange := &model.Exchange{ID: 1, Name: "Mexc"}
	if err := exchangeStore.CreateExchange(exchange); err != nil {
		t.Fatalf("failed to create exchange: %v", err)
	}

	apiKey := "test-api-key"
	apiSecret := "test-api-secret"

	encryptedKey, err := security.EncryptString(apiKey)
	if err != nil {
		t.Fatalf("failed to encrypt api key: %v", err)
	}
	encryptedSecret, err := security.EncryptString(apiSecret)
	if err != nil {
		t.Fatalf("failed to encrypt api secret: %v", err)
	}

	userExchange := &model.UserExchange{
		UserID:        42,
		ExchangeID:    exchange.ID,
		APIKeyHash:    encryptedKey,
		APISecretHash: encryptedSecret,
		Exchange:      exchange,
	}

	if err := exchangeStore.SaveUserExchange(userExchange); err != nil {
		t.Fatalf("failed to save user exchange: %v", err)
	}

	mockConnector := &mockMexcConnector{}
	originalFactory := mexcConnectorFactory
	var receivedKey, receivedSecret string
	mexcConnectorFactory = func(key, secret string) mexcConnector {
		receivedKey = key
		receivedSecret = secret
		return mockConnector
	}
	t.Cleanup(func() {
		mexcConnectorFactory = originalFactory
	})

	payload := map[string]string{
		"apiKey":    apiKey,
		"apiSecret": apiSecret,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/user-exchanges/1/test", bytes.NewReader(body))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("exchangeID", strconv.Itoa(int(exchange.ID)))
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)
	ctx = context.WithValue(ctx, auth.UserKey, &model.User{ID: userExchange.UserID})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler := TestMexcConnectionHandler(logger)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var response map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", response["status"])
	}

	if mockConnector.called != 1 {
		t.Fatalf("expected connector to be called once, got %d", mockConnector.called)
	}

	if receivedKey != apiKey || receivedSecret != apiSecret {
		t.Fatalf("expected connector to receive provided credentials")
	}
}

func TestTestMexcConnectionHandlerCredentialMismatch(t *testing.T) {
	setEncryptionKeyForTest(t)

	logger := logrus.NewEntry(logrus.StandardLogger())

	exchangeStore := newInMemoryUserExchangeStore()
	repository.SetUserExchangeStore(exchangeStore)
	t.Cleanup(func() {
		repository.SetUserExchangeStore(nil)
	})

	exchange := &model.Exchange{ID: 1, Name: "Mexc"}
	if err := exchangeStore.CreateExchange(exchange); err != nil {
		t.Fatalf("failed to create exchange: %v", err)
	}

	encryptedKey, err := security.EncryptString("stored-key")
	if err != nil {
		t.Fatalf("failed to encrypt api key: %v", err)
	}
	encryptedSecret, err := security.EncryptString("stored-secret")
	if err != nil {
		t.Fatalf("failed to encrypt api secret: %v", err)
	}

	userExchange := &model.UserExchange{
		UserID:        99,
		ExchangeID:    exchange.ID,
		APIKeyHash:    encryptedKey,
		APISecretHash: encryptedSecret,
		Exchange:      exchange,
	}

	if err := exchangeStore.SaveUserExchange(userExchange); err != nil {
		t.Fatalf("failed to save user exchange: %v", err)
	}

	mockConnector := &mockMexcConnector{}
	originalFactory := mexcConnectorFactory
	var receivedKey, receivedSecret string
	mexcConnectorFactory = func(key, secret string) mexcConnector {
		receivedKey = key
		receivedSecret = secret
		return mockConnector
	}
	t.Cleanup(func() {
		mexcConnectorFactory = originalFactory
	})

	payload := map[string]string{
		"apiKey":    "wrong-key",
		"apiSecret": "stored-secret",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/user-exchanges/1/test", bytes.NewReader(body))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("exchangeID", strconv.Itoa(int(exchange.ID)))
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)
	ctx = context.WithValue(ctx, auth.UserKey, &model.User{ID: userExchange.UserID})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler := TestMexcConnectionHandler(logger)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 despite credential mismatch, got %d", rec.Code)
	}

	if receivedKey != "stored-key" || receivedSecret != "stored-secret" {
		t.Fatalf("expected connector to receive stored credentials, got key=%q secret=%q", receivedKey, receivedSecret)
	}
}

func TestTestMexcConnectionHandlerUnsupportedExchange(t *testing.T) {
	setEncryptionKeyForTest(t)

	logger := logrus.NewEntry(logrus.StandardLogger())

	exchangeStore := newInMemoryUserExchangeStore()
	repository.SetUserExchangeStore(exchangeStore)
	t.Cleanup(func() {
		repository.SetUserExchangeStore(nil)
	})

	exchange := &model.Exchange{ID: 1, Name: "Kucoin"}
	if err := exchangeStore.CreateExchange(exchange); err != nil {
		t.Fatalf("failed to create exchange: %v", err)
	}

	encryptedKey, err := security.EncryptString("stored-key")
	if err != nil {
		t.Fatalf("failed to encrypt api key: %v", err)
	}
	encryptedSecret, err := security.EncryptString("stored-secret")
	if err != nil {
		t.Fatalf("failed to encrypt api secret: %v", err)
	}

	userExchange := &model.UserExchange{
		UserID:        77,
		ExchangeID:    exchange.ID,
		APIKeyHash:    encryptedKey,
		APISecretHash: encryptedSecret,
		Exchange:      exchange,
	}

	if err := exchangeStore.SaveUserExchange(userExchange); err != nil {
		t.Fatalf("failed to save user exchange: %v", err)
	}

	payload := map[string]string{
		"apiKey":    "stored-key",
		"apiSecret": "stored-secret",
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/user-exchanges/1/test", bytes.NewReader(body))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("exchangeID", strconv.Itoa(int(exchange.ID)))
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)
	ctx = context.WithValue(ctx, auth.UserKey, &model.User{ID: userExchange.UserID})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler := TestMexcConnectionHandler(logger)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400 for unsupported exchange, got %d", rec.Code)
	}
}

func TestTestMexcConnectionHandlerConnectorFailure(t *testing.T) {
	setEncryptionKeyForTest(t)

	logger := logrus.NewEntry(logrus.StandardLogger())

	exchangeStore := newInMemoryUserExchangeStore()
	repository.SetUserExchangeStore(exchangeStore)
	t.Cleanup(func() {
		repository.SetUserExchangeStore(nil)
	})

	exchange := &model.Exchange{ID: 1, Name: "Mexc"}
	if err := exchangeStore.CreateExchange(exchange); err != nil {
		t.Fatalf("failed to create exchange: %v", err)
	}

	apiKey := "test-api-key"
	apiSecret := "test-api-secret"

	encryptedKey, err := security.EncryptString(apiKey)
	if err != nil {
		t.Fatalf("failed to encrypt api key: %v", err)
	}
	encryptedSecret, err := security.EncryptString(apiSecret)
	if err != nil {
		t.Fatalf("failed to encrypt api secret: %v", err)
	}

	userExchange := &model.UserExchange{
		UserID:        55,
		ExchangeID:    exchange.ID,
		APIKeyHash:    encryptedKey,
		APISecretHash: encryptedSecret,
		Exchange:      exchange,
	}

	if err := exchangeStore.SaveUserExchange(userExchange); err != nil {
		t.Fatalf("failed to save user exchange: %v", err)
	}

	mockConnector := &mockMexcConnector{err: errors.New("ping failed")}
	originalFactory := mexcConnectorFactory
	mexcConnectorFactory = func(key, secret string) mexcConnector {
		return mockConnector
	}
	t.Cleanup(func() {
		mexcConnectorFactory = originalFactory
	})

	payload := map[string]string{
		"apiKey":    apiKey,
		"apiSecret": apiSecret,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/user-exchanges/1/test", bytes.NewReader(body))
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("exchangeID", strconv.Itoa(int(exchange.ID)))
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx)
	ctx = context.WithValue(ctx, auth.UserKey, &model.User{ID: userExchange.UserID})
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	handler := TestMexcConnectionHandler(logger)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502 when connector fails, got %d", rec.Code)
	}
}
