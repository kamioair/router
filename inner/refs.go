package main

type refStruct struct{}

var refs refStruct

func (ref *refStruct) newDeviceCode() (string, error) {
	param := map[string]any{
		"IsRoot": service.IsRoot(),
	}
	ctx, err := service.SendRequest("ClientManager", "NewDeviceCode", param)
	if err != nil {
		return "", err
	}
	return ctx.Raw().(string), nil
}

func (ref *refStruct) knockDoor(info map[string]string) (any, error) {
	ctx, err := service.SendRequest("ClientManager", "KnockDoor", info)
	if err != nil {
		return "", err
	}
	return ctx.Raw(), nil
}
