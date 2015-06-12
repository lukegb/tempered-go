package tempered-go

// #cgo LDFLAGS: -ltempered -lhidapi-hidraw
// #include <tempered.h>
// #include <stdlib.h>
import "C"

import (
	"errors"
	"log"
	"unsafe"
)

var (
	ERR_NOT_INITED      = errors.New(`tempered: not initialised`)
	ERR_NOT_OPEN        = errors.New(`tempered: device not open`)
	ERR_FAILED_RETRIEVE = errors.New(`tempered: failed to retrieve sensor reading`)
	ERR_FAILED_UPDATE   = errors.New(`tempered: failed to update sensors`)
)

type Tempered struct {
	inited bool
}

type TemperedDevice struct {
	dev unsafe.Pointer

	Path            string
	TypeName        string
	VendorId        uint
	ProductId       uint
	InterfaceNumber int
}

type TemperedSensorType int

func (st TemperedSensorType) IsType(t TemperedSensorType) bool {
	return st&t == t
}

const (
	TEMPERED_SENSOR_TYPE_TEMPERATURE = C.TEMPERED_SENSOR_TYPE_TEMPERATURE
	TEMPERED_SENSOR_TYPE_HUMIDITY    = C.TEMPERED_SENSOR_TYPE_HUMIDITY
)

type TemperedSensor struct {
	device    *TemperedDevice
	sensorNum int

	TypeMask TemperedSensorType
}

func (ts *TemperedSensor) Temperature() (float64, error) {
	return ts.device.Temperature(ts.sensorNum)
}

func (ts *TemperedSensor) Humidity() (float64, error) {
	return ts.device.Humidity(ts.sensorNum)
}

func (t *TemperedDevice) Open() error {
	if t.dev != nil {
		return nil
	}

	devList := C.struct_tempered_device_list{
		next:             nil,
		path:             C.CString(t.Path),
		type_name:        C.CString(t.TypeName),
		vendor_id:        C.ushort(t.VendorId),
		product_id:       C.ushort(t.ProductId),
		interface_number: C.int(t.InterfaceNumber),
	}
	defer func() {
		C.free(unsafe.Pointer(devList.path))
		C.free(unsafe.Pointer(devList.type_name))
	}()

	var errCstr *C.char
	devRet := C.tempered_open(&devList, &errCstr)
	if devRet == nil {
		err := errors.New(C.GoString(errCstr))
		C.free(unsafe.Pointer(errCstr))
		return err
	}

	t.dev = unsafe.Pointer(devRet)

	return nil
}

func (t *TemperedDevice) getParamDev() *C.struct_tempered_device_ {
	return (*C.struct_tempered_device_)(unsafe.Pointer(t.dev))
}

func (t *TemperedDevice) SensorCount() (int, error) {
	if t.dev == nil {
		return 0, ERR_NOT_OPEN
	}

	sCount := int(C.tempered_get_sensor_count(t.getParamDev()))

	return sCount, nil
}

func (t *TemperedDevice) Update() error {
	if t.dev == nil {
		return ERR_NOT_OPEN
	}

	didWork := C.tempered_read_sensors(t.getParamDev())

	if !didWork {
		return ERR_FAILED_UPDATE
	}
	return nil
}

func (t *TemperedDevice) Sensors() ([]*TemperedSensor, error) {
	if t.dev == nil {
		return nil, ERR_NOT_OPEN
	}

	tsList := []*TemperedSensor{}
	sCount := int(C.tempered_get_sensor_count(t.getParamDev()))
	for n := 0; n < sCount; n++ {
		ts := new(TemperedSensor)
		ts.device = t
		ts.sensorNum = n
		ts.TypeMask = TemperedSensorType(C.tempered_get_sensor_type(t.getParamDev(), C.int(n)))
		tsList = append(tsList, ts)
	}

	return tsList, nil
}

func (t *TemperedDevice) Temperature(sensorNum int) (float64, error) {
	if t.dev == nil {
		return 0, ERR_NOT_OPEN
	}

	var cFloat C.float
	retrOk := C.tempered_get_temperature(t.getParamDev(), C.int(sensorNum), &cFloat)
	if !retrOk {
		return 0, ERR_FAILED_RETRIEVE
	}

	return float64(cFloat), nil
}

func (t *TemperedDevice) Humidity(sensorNum int) (float64, error) {
	if t.dev == nil {
		return 0, ERR_NOT_OPEN
	}

	var cFloat C.float
	retrOk := C.tempered_get_humidity(t.getParamDev(), C.int(sensorNum), &cFloat)
	if !retrOk {
		return 0, ERR_FAILED_RETRIEVE
	}

	return float64(cFloat), nil
}

func (t *TemperedDevice) Close() error {
	if t.dev == nil {
		return nil
	}

	C.tempered_close(t.getParamDev())
	return nil
}

func (t *Tempered) Init() error {
	if t.inited {
		return nil
	}

	var errCstr *C.char
	ret := C.tempered_init(&errCstr)
	if !ret {
		err := errors.New(C.GoString(errCstr))
		C.free(unsafe.Pointer(errCstr))
		return err
	}

	t.inited = true
	return nil
}

func (t *Tempered) DeviceList() ([]TemperedDevice, error) {
	if !t.inited {
		return nil, ERR_NOT_INITED
	}

	var errCstr *C.char
	var cDevices *C.struct_tempered_device_list
	cDevices = C.tempered_enumerate(&errCstr)
	if cDevices == nil {
		err := errors.New(C.GoString(errCstr))
		C.free(unsafe.Pointer(errCstr))
		return nil, err
	}
	defer func() {
		C.tempered_free_device_list(cDevices)
	}()

	tds := []TemperedDevice{}
	for dev := cDevices; dev != nil; dev = dev.next {
		td := TemperedDevice{
			Path:            C.GoString(dev.path),
			TypeName:        C.GoString(dev.type_name),
			VendorId:        uint(dev.vendor_id),
			ProductId:       uint(dev.product_id),
			InterfaceNumber: int(dev.interface_number),
		}
		tds = append(tds, td)
	}

	return tds, nil
}

func (t *Tempered) Exit() error {
	if !t.inited {
		return nil
	}

	var errCstr *C.char
	ret := C.tempered_exit(&errCstr)
	if !ret {
		err := errors.New(C.GoString(errCstr))
		C.free(unsafe.Pointer(errCstr))
		return err
	}

	t.inited = false
	return nil
}

/*
func main() {
	log.Println("TEMPERED")
	t := new(Tempered)
	log.Println("init", t.Init())
	log.Println(t)
	td, err := t.DeviceList()
	for _, dev := range td {
		if err := dev.Open(); err != nil {
			log.Println("err opening device", dev, err)
			continue
		}
		if err := dev.Update(); err != nil {
			log.Println(dev, "err updating", err)
			dev.Close()
			continue
		}
		if sensors, err := dev.Sensors(); err != nil {
			log.Println(dev, "err getting sensors", err)
		} else {
			log.Println(dev, "got sensors", len(sensors))
			for _, sensor := range sensors {
				log.Println("\t", sensor)
				if sensor.TypeMask.IsType(TEMPERED_SENSOR_TYPE_TEMPERATURE) {
					val, err := sensor.Temperature()
					log.Println("\t\t", "temperature", val, err)
				}
				if sensor.TypeMask.IsType(TEMPERED_SENSOR_TYPE_HUMIDITY) {
					val, err := sensor.Humidity()
					log.Println("\t\t", "humidity", val, err)
				}
			}
		}
		dev.Close()
	}
	log.Println("devicelist", td, err)
	log.Println("exit", t.Exit())
}
*/
