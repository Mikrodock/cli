package provision

import (
	"fmt"
	"mikrodock-cli/utils"
)

func init() {
	ProviderFactoryRegister(UbuntuProviderFactory)
}

func UbuntuProviderFactory() Provider {
	prov := &UbuntuProvider{}
	return prov
}

type ProviderFactory func() Provider

var providerFactory = make([]ProviderFactory, 0)

func ProviderFactoryRegister(provider ProviderFactory) {
	providerFactory = append(providerFactory, provider)
}

func GetMatchingProvider(ostype utils.OSType) (Provider, error) {
	for _, provFact := range providerFactory {
		provider := provFact()
		if provider.MatchOS(ostype) {
			return provider, nil
		}

	}
	return nil, fmt.Errorf("Cannot find provider for OSType %s", ostype)
}

// func NewDriver(name string, conf map[string]string) (InitDriver, error) {
// 	FacFunc := driverFactory[name]
// 	if FacFunc == nil {
// 		return nil, fmt.Errorf("No factory named %s", name)
// 	}
// 	return FacFunc(conf), nil
// }
