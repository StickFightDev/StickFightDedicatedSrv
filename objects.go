package main

//SyncableObject holds an object to sync across clients and interact with
type SyncableObject struct {
	PositionX       float32 //The X position of the object
	PositionY       float32 //The Y position of the object
	RotationX       float32 //The X rotation of the object
	RotationY       float32 //The Y rotation of the object
	ScaleX          float32 //The X scale of the object
	ScaleY          float32 //The Y scale of the object
	ObjectID        string  //The ID of the object type
	HasMirrorObject bool    //If the object should be spawned with a mirror object too
	PropsSeed       int     //???
	NetworkID       int     //The object sync ID to track this object
}

//SyncableWeapon holds a weapon object to sync across clients and interact with
type SyncableWeapon struct {
	PositionX       float32 //The X position of the weapon
	PositionY       float32 //The Y position of the weapon
	WeaponID        int     //The ID of the weapon type
	HasMirrorObject bool    //If the weapon should be spawned with a mirror weapon too
	NetworkID       int     //The object sync ID to track this weapon
}
