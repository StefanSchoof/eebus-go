package usecase

import (
	"errors"
	"fmt"

	"github.com/DerAndereAndi/eebus-go/service"
	"github.com/DerAndereAndi/eebus-go/spine"
	"github.com/DerAndereAndi/eebus-go/spine/model"
	"github.com/DerAndereAndi/eebus-go/util"
)

// Interface for the evseCC use case
type evseCCDelegate interface {
	// handle device state updates from the remote device
	HandleDeviceState(ski string, failure bool, errorCode string)
}

// EVSE Commissioning and Configuration Use Case implementation
// Important notes:
// The use case specification only defines the EVSE as providing error state update,
// but in reality, both actors do.
// Also the CEM needs to provide a heartbeat as part of the DeviceDiagnosis feature,
// but the spec lacks a use case where this should be done. So we do this in here as well.
type evseCC struct {
	*UseCaseImpl
	service *service.EEBUSService

	// only required by CEM
	Delegate evseCCDelegate
}

// Register the use case
func RegisterEvseCC(service *service.EEBUSService) evseCC {
	entity := service.LocalEntity()

	// add the use case
	useCase := &evseCC{
		UseCaseImpl: NewUseCase(
			entity,
			model.UseCaseNameTypeEVSECommissioningAndConfiguration,
			[]model.UseCaseScenarioSupportType{1, 2}),
		service: service,
	}

	// both actors need to subscribe, instead of only the CEM as the spec defines
	spine.Events.Subscribe(useCase)

	// add the features
	{
		f := spine.NewFeatureLocalImpl(entity.NextFeatureId(), entity, model.FeatureTypeTypeDeviceClassification, model.RoleTypeClient)
		f.SetDescriptionString("Device Classification Client")

		entity.AddFeature(f)
	}

	// both actors need a client and a server role feature for DeviceDiagnosis
	{
		f := spine.NewFeatureLocalImpl(entity.NextFeatureId(), entity, model.FeatureTypeTypeDeviceDiagnosis, model.RoleTypeServer)
		f.SetDescriptionString("Device Diagnosis Server")

		// Set the initial state
		deviceDiagnosisStateDate := &model.DeviceDiagnosisStateDataType{
			OperatingState: util.Ptr(model.DeviceDiagnosisOperatingStateTypeNormalOperation),
		}
		f.SetData(model.FunctionTypeDeviceDiagnosisStateData, deviceDiagnosisStateDate)

		entity.AddFeature(f)
	}
	{
		f := spine.NewFeatureLocalImpl(entity.NextFeatureId(), entity, model.FeatureTypeTypeDeviceDiagnosis, model.RoleTypeClient)
		f.SetDescriptionString("Device Diagnosis Client")
		entity.AddFeature(f)
	}

	return *useCase
}

// public method to allow updating the device state
// this will be sent to all remote devices
func (r *evseCC) UpdateErrorState(failure bool, errorCode string) {
	deviceDiagnosisStateDate := &model.DeviceDiagnosisStateDataType{}
	if failure {
		deviceDiagnosisStateDate.OperatingState = util.Ptr(model.DeviceDiagnosisOperatingStateTypeFailure)
		deviceDiagnosisStateDate.LastErrorCode = util.Ptr(model.LastErrorCodeType(errorCode))
	} else {
		deviceDiagnosisStateDate.OperatingState = util.Ptr(model.DeviceDiagnosisOperatingStateTypeNormalOperation)
	}
	r.notifyDeviceDiagnosisState(deviceDiagnosisStateDate)
}

// Internal EventHandler Interface
func (r *evseCC) HandleEvent(payload spine.EventPayload) {
	switch payload.EventType {
	case spine.EventTypeDeviceChange:
		if payload.ChangeType == spine.ElementChangeAdd {
			r.requestManufacturer(payload.Device)
			r.requestDeviceDiagnosisState(payload.Device)
		}
	case spine.EventTypeDataChange:
		if payload.ChangeType == spine.ElementChangeUpdate {
			switch payload.Data.(type) {
			case *model.DeviceDiagnosisStateDataType:
				if r.Delegate == nil {
					return
				}

				deviceDiagnosisStateData := payload.Data.(model.DeviceDiagnosisStateDataType)
				failure := *deviceDiagnosisStateData.OperatingState == model.DeviceDiagnosisOperatingStateTypeFailure
				r.Delegate.HandleDeviceState(payload.Ski, failure, string(*deviceDiagnosisStateData.LastErrorCode))
			}
		}
	}
}

// request DeviceClassificationManufacturerData from a remote device
func (r *evseCC) requestManufacturer(remoteDevice *spine.DeviceRemoteImpl) {
	featureLocal, featureRemote, err := r.getLocalClientAndRemoteServerFeatures(model.FeatureTypeTypeDeviceClassification, remoteDevice)

	if err != nil {
		fmt.Println(err)
		return
	}

	requestChannel := make(chan *model.DeviceClassificationManufacturerDataType)
	featureLocal.RequestData(model.FunctionTypeDeviceClassificationManufacturerData, featureRemote, requestChannel)
}

// request DeviceDiagnosisStateData from a remote device
func (r *evseCC) requestDeviceDiagnosisState(remoteDevice *spine.DeviceRemoteImpl) {
	featureLocal, featureRemote, err := r.getLocalClientAndRemoteServerFeatures(model.FeatureTypeTypeDeviceDiagnosis, remoteDevice)

	if err != nil {
		fmt.Println(err)
		return
	}

	requestChannel := make(chan *model.DeviceDiagnosisStateDataType)
	featureLocal.RequestData(model.FunctionTypeDeviceDiagnosisStateData, featureRemote, requestChannel)

	// subscribe to device diagnosis state updates
	remoteDevice.Sender().Subscribe(featureLocal.Address(), featureRemote.Address(), model.FeatureTypeTypeDeviceDiagnosis)
}

// notify remote devices about the new device diagnosis state
func (r *evseCC) notifyDeviceDiagnosisState(operatingState *model.DeviceDiagnosisStateDataType) {
	for _, remoteDevice := range r.service.RemoteDevices() {
		featureLocal, featureRemote, err := r.getLocalClientAndRemoteServerFeatures(model.FeatureTypeTypeDeviceDiagnosis, remoteDevice)

		if err != nil {
			fmt.Println(err)
			continue
		}

		featureLocal.SetData(model.FunctionTypeDeviceDiagnosisStateData, operatingState)

		featureLocal.NotifyData(model.FunctionTypeDeviceDiagnosisStateData, featureRemote)
	}
}

// internal helper method for getting local and remote feature for a given featureType and a given remoteDevice
func (r *evseCC) getLocalClientAndRemoteServerFeatures(featureType model.FeatureTypeType, remoteDevice *spine.DeviceRemoteImpl) (spine.FeatureLocal, *spine.FeatureRemoteImpl, error) {
	featureLocal := r.Entity.Device().FeatureByTypeAndRole(featureType, model.RoleTypeClient)
	featureRemote := remoteDevice.FeatureByTypeAndRole(featureType, model.RoleTypeServer)

	if featureLocal == nil || featureRemote == nil {
		return nil, nil, errors.New("local or remote feature not found")
	}

	return featureLocal, featureRemote, nil
}