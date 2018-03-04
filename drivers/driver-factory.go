package drivers

import (
	"fmt"
)

func init() {
	DriverFactoryRegister("digitalocean", DigitalOceanFactory)
}

type InitDriver func(map[string]interface{}) (Driver, error)

func DigitalOceanFactory(conf map[string]string) InitDriver {

	return func(instanceConf map[string]interface{}) (Driver, error) {
		d := &DigitalOceanDriver{
			AccessToken: conf["access-token"],
		}
		err := d.PreCreate(instanceConf)
		if err != nil {
			return nil, err
		}
		return d, err
	}

}

type DriverFactory func(conf map[string]string) InitDriver

var driverFactory = make(map[string]DriverFactory)

func DriverFactoryRegister(name string, driver DriverFactory) {
	driverFactory[name] = driver
}

func NewDriver(name string, conf map[string]string) (InitDriver, error) {
	FacFunc := driverFactory[name]
	if FacFunc == nil {
		return nil, fmt.Errorf("No factory named %s", name)
	}
	return FacFunc(conf), nil
}
