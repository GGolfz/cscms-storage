package data

import (
	"github.com/go-test/deep"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/thetkpark/cscms-temp-storage/data/model"
	"gorm.io/gorm"
	"testing"
)

type GormImageDataStoreTestSuite struct {
	suite.Suite
	db        *gorm.DB
	store     *GormImageDataStore
	user      *model.User
	image     *model.Image
	ownImages []model.Image
}

func TestNewGormImageDataStore(t *testing.T) {
	db, err := createTestGormDB()
	require.NoError(t, err)
	store, err := NewGormImageDataStore(db)
	require.NoError(t, err)
	require.NotNil(t, store)

	require.NoError(t, db.Create(createTestImage(0)).Error)
	require.NoError(t, destroyTestGormDB())
}

func TestGormImageDataStore(t *testing.T) {
	suite.Run(t, new(GormImageDataStoreTestSuite))
}

func (s *GormImageDataStoreTestSuite) SetupTest() {
	gormDB, err := createTestGormDB()
	require.NoError(s.T(), err)
	s.db = gormDB

	require.NoError(s.T(), gormDB.AutoMigrate(&model.Image{}, &model.User{}))

	s.store = &GormImageDataStore{db: gormDB}
	require.NoError(s.T(), err)

	s.user = createTestUser("github")
	s.image = createTestImage(0)
	s.ownImages = []model.Image{
		*createTestImage(s.user.ID),
		*createTestImage(s.user.ID),
		*createTestImage(s.user.ID),
	}
	require.NoError(s.T(), s.db.Create(s.user).Error)
	require.NoError(s.T(), s.db.Create(s.image).Error)
	require.NoError(s.T(), s.db.Create(&s.ownImages).Error)
}

func (s *GormImageDataStoreTestSuite) AfterTest(_, _ string) {
	require.NoError(s.T(), destroyTestGormDB())
}

func (s *GormImageDataStoreTestSuite) TestCreate() {
	newImage := createTestImage(0)
	require.NoError(s.T(), s.store.Create(newImage))

	var queryImage model.Image
	require.NoError(s.T(), s.db.Where(newImage).First(&queryImage).Error)
	require.Nil(s.T(), deep.Equal(&queryImage, newImage))
}

func (s *GormImageDataStoreTestSuite) TestFindByID() {
	queryImage, err := s.store.FindByID(s.image.ID)
	require.NoError(s.T(), err)
	require.Nil(s.T(), deep.Equal(queryImage, s.image))
}

func (s *GormImageDataStoreTestSuite) TestFindByIDNotFound() {
	newImg := createTestImage(0)
	queryImage, err := s.store.FindByID(newImg.ID)
	require.NoError(s.T(), err)
	require.Nil(s.T(), queryImage)
}

func (s *GormImageDataStoreTestSuite) TestFindByUserID() {
	images, err := s.store.FindByUserID(s.user.ID)
	require.NoError(s.T(), err)
	require.Len(s.T(), *images, len(s.ownImages))
}

func (s *GormImageDataStoreTestSuite) TestFindByUserIDEmpty() {
	newUser := createTestUser("google")
	images, err := s.store.FindByUserID(newUser.ID)
	require.NoError(s.T(), err)
	require.Len(s.T(), *images, 0)
}

func (s *GormImageDataStoreTestSuite) TestDeleteByID() {
	require.NoError(s.T(), s.store.DeleteByID(s.image.ID))

	var queryImage model.Image
	require.ErrorIs(s.T(), s.db.Where(s.image).First(&queryImage).Error, gorm.ErrRecordNotFound)
}
