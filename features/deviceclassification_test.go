package features

import (
	"testing"

	"github.com/enbility/eebus-go/spine"
	"github.com/enbility/eebus-go/spine/model"
	"github.com/enbility/eebus-go/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestDeviceClassificationSuite(t *testing.T) {
	suite.Run(t, new(DeviceClassificationSuite))
}

type DeviceClassificationSuite struct {
	suite.Suite

	localDevice  *spine.DeviceLocalImpl
	remoteEntity *spine.EntityRemoteImpl

	deviceClassification *DeviceClassification
	sentMessage          []byte
}

var _ spine.SpineDataConnection = (*DeviceClassificationSuite)(nil)

func (s *DeviceClassificationSuite) WriteSpineMessage(message []byte) {
	s.sentMessage = message
}

func (s *DeviceClassificationSuite) BeforeTest(suiteName, testName string) {
	s.localDevice, s.remoteEntity = setupFeatures(
		s.T(),
		s,
		[]featureFunctions{
			{
				featureType: model.FeatureTypeTypeDeviceClassification,
				functions: []model.FunctionType{
					model.FunctionTypeDeviceClassificationManufacturerData,
				},
			},
		},
	)

	var err error
	s.deviceClassification, err = NewDeviceClassification(model.RoleTypeServer, model.RoleTypeClient, s.localDevice, s.remoteEntity)
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), s.deviceClassification)
}

func (s *DeviceClassificationSuite) Test_RequestManufacturerDetails() {
	counter, err := s.deviceClassification.RequestManufacturerDetails()
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), counter)
}

func (s *DeviceClassificationSuite) Test_GetManufacturerDetails() {
	result, err := s.deviceClassification.GetManufacturerDetails()
	assert.NotNil(s.T(), err)
	assert.Nil(s.T(), result)

	rF := s.remoteEntity.Feature(util.Ptr(model.AddressFeatureType(1)))
	fData := &model.DeviceClassificationManufacturerDataType{
		DeviceName:                     util.Ptr(model.DeviceClassificationStringType("brand")),
		DeviceCode:                     util.Ptr(model.DeviceClassificationStringType("brand")),
		SerialNumber:                   util.Ptr(model.DeviceClassificationStringType("brand")),
		SoftwareRevision:               util.Ptr(model.DeviceClassificationStringType("brand")),
		HardwareRevision:               util.Ptr(model.DeviceClassificationStringType("brand")),
		VendorName:                     util.Ptr(model.DeviceClassificationStringType("brand")),
		VendorCode:                     util.Ptr(model.DeviceClassificationStringType("brand")),
		BrandName:                      util.Ptr(model.DeviceClassificationStringType("brand")),
		PowerSource:                    util.Ptr(model.PowerSourceType("brand")),
		ManufacturerNodeIdentification: util.Ptr(model.DeviceClassificationStringType("brand")),
		ManufacturerLabel:              util.Ptr(model.LabelType("label")),
		ManufacturerDescription:        util.Ptr(model.DescriptionType("description")),
	}
	rF.UpdateData(model.FunctionTypeDeviceClassificationManufacturerData, fData, nil, nil)

	result, err = s.deviceClassification.GetManufacturerDetails()
	assert.Nil(s.T(), err)
	assert.NotNil(s.T(), result)
}
