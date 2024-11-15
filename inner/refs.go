package main

type refStruct struct{}

var refs refStruct

func (ref *refStruct) newDeviceCode() (string, error) {
	ctx, err := service.SendRequest("ClientManager", "NewDeviceCode", nil)
	if err != nil {
		return "", err
	}
	return ctx.Raw().(string), nil
}
