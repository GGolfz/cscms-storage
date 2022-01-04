package data

import (
	"github.com/bxcodec/faker/v3"
	"github.com/go-test/deep"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/thetkpark/cscms-temp-storage/data/model"
	"gorm.io/gorm"
	"testing"
)

type GormUserDataStoreTestSuite struct {
	suite.Suite
	db    *gorm.DB
	store *GormUserDataStore
	user  *model.User
}

func TestNewGormUserDataStore(t *testing.T) {
	db, err := createTestGormDB()
	require.NoError(t, err)
	store, err := NewGormUserDataStore(db)
	require.NoError(t, err)
	require.NotNil(t, store)

	require.NoError(t, db.Create(createTestUser("github")).Error)
	require.NoError(t, destroyTestGormDB())
}

func TestGormUserDataStore(t *testing.T) {
	suite.Run(t, new(GormUserDataStoreTestSuite))
}

func (s *GormUserDataStoreTestSuite) SetupTest() {
	gormDB, err := createTestGormDB()
	require.NoError(s.T(), err)
	s.db = gormDB

	require.NoError(s.T(), gormDB.AutoMigrate(&model.User{}))

	s.store = &GormUserDataStore{db: gormDB}
	require.NoError(s.T(), err)

	s.user = createTestUser("github")
	require.NoError(s.T(), gormDB.Create(s.user).Error)
	require.NoError(s.T(), gormDB.Create(createTestUser("github")).Error)
	require.NoError(s.T(), gormDB.Create(createTestUser("google")).Error)
}

func (s *GormUserDataStoreTestSuite) AfterTest(_, _ string) {
	require.NoError(s.T(), destroyTestGormDB())
}

func (s *GormUserDataStoreTestSuite) TestCreate() {
	newUser := createTestUser("google")
	user, err := s.store.Create(newUser.Email, newUser.Username, newUser.Provider, newUser.AvatarURL)
	require.NoError(s.T(), err)

	var queryUser model.User
	require.NoError(s.T(), s.db.Where(user).First(&queryUser).Error)
	require.Nil(s.T(), deep.Equal(user, &queryUser))
}

func (s *GormUserDataStoreTestSuite) TestFoundByID() {
	foundUser, err := s.store.FindById(s.user.ID)
	require.NoError(s.T(), err)
	require.Nil(s.T(), deep.Equal(foundUser, s.user))
}

func (s *GormUserDataStoreTestSuite) TestNotFoundByID() {
	newUser := createTestUser("github")
	foundUser, err := s.store.FindById(newUser.ID)
	require.NoError(s.T(), err)
	require.Nil(s.T(), foundUser)
}

func (s *GormUserDataStoreTestSuite) TestFoundByProviderAndEmail() {
	foundUser, err := s.store.FindByProviderAndEmail(s.user.Provider, s.user.Email)
	require.NoError(s.T(), err)
	require.Nil(s.T(), deep.Equal(foundUser, s.user))
}

func (s *GormUserDataStoreTestSuite) TestNotFoundByProviderAndEmail() {
	newUser := createTestUser("github")
	foundUser, err := s.store.FindByProviderAndEmail(newUser.Provider, newUser.Email)
	require.NoError(s.T(), err)
	require.Nil(s.T(), foundUser)
}

func (s *GormUserDataStoreTestSuite) TestFoundByAPIKey() {
	foundUser, err := s.store.FindByAPIKey(s.user.APIKey)
	require.NoError(s.T(), err)
	require.Nil(s.T(), deep.Equal(foundUser, s.user))
}

func (s *GormUserDataStoreTestSuite) TestNotFoundByAPIKey() {
	newUser := createTestUser("github")
	foundUser, err := s.store.FindByAPIKey(newUser.APIKey)
	require.NoError(s.T(), err)
	require.Nil(s.T(), foundUser)
}

func (s *GormUserDataStoreTestSuite) TestUpdateAPIKey() {
	newApiKey := faker.UUIDDigit()
	require.NoError(s.T(), s.store.UpdateAPIKey(s.user.ID, newApiKey))

	queryUser := &model.User{}
	err := s.db.Where(&model.User{APIKey: newApiKey}).First(queryUser).Error
	require.NoError(s.T(), err)
	require.Equal(s.T(), queryUser.ID, s.user.ID)
}

func (s *GormUserDataStoreTestSuite) TestUpdateAPIKeyOnNotFoundUser() {
	newUser := createTestUser("github")
	newApiKey := faker.UUIDDigit()
	require.NoError(s.T(), s.store.UpdateAPIKey(newUser.ID, newApiKey))

	queryUser := &model.User{}
	err := s.db.Where(&model.User{APIKey: newApiKey}).First(queryUser).Error
	require.ErrorIs(s.T(), err, gorm.ErrRecordNotFound)
}
