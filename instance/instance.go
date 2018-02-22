package instance

import (
	"github.com/dropbox/godropbox/container/set"
	"github.com/dropbox/godropbox/errors"
	"github.com/pritunl/pritunl-cloud/bridge"
	"github.com/pritunl/pritunl-cloud/database"
	"github.com/pritunl/pritunl-cloud/disk"
	"github.com/pritunl/pritunl-cloud/errortypes"
	"github.com/pritunl/pritunl-cloud/vm"
	"gopkg.in/mgo.v2/bson"
	"strconv"
)

type Instance struct {
	Id           bson.ObjectId `bson:"_id,omitempty" json:"id"`
	Organization bson.ObjectId `bson:"organization,omitempty" json:"organization"`
	Zone         bson.ObjectId `bson:"zone,omitempty" json:"zone"`
	Image        bson.ObjectId `bson:"image,omitempty" json:"image"`
	Status       string        `bson:"-" json:"status"`
	State        string        `bson:"state" json:"state"`
	VmState      string        `bson:"vm_state" json:"vm_state"`
	PublicIp     string        `bson:"public_ip" json:"public_ip"`
	PublicIp6    string        `bson:"public_ip6" json:"public_ip6"`
	Node         bson.ObjectId `bson:"node,omitempty" json:"node"`
	Name         string        `bson:"name" json:"name"`
	Memory       int           `bson:"memory" json:"memory"`
	Processors   int           `bson:"processors" json:"processors"`
	NetworkRoles []string      `bson:"network_roles" json:"network_roles"`
}

func (i *Instance) Validate(db *database.Database) (
	errData *errortypes.ErrorData, err error) {

	if i.State == "" {
		i.State = Running
	}

	if i.Organization == "" {
		errData = &errortypes.ErrorData{
			Error:   "organization_required",
			Message: "Missing required organization",
		}
	}

	if i.Zone == "" {
		errData = &errortypes.ErrorData{
			Error:   "zone_required",
			Message: "Missing required zone",
		}
	}

	if i.Node == "" {
		errData = &errortypes.ErrorData{
			Error:   "node_required",
			Message: "Missing required node",
		}
	}

	if i.Image == "" {
		errData = &errortypes.ErrorData{
			Error:   "image_required",
			Message: "Missing required image",
		}
	}

	if i.Memory < 256 {
		i.Memory = 256
	}

	if i.Processors < 1 {
		i.Processors = 1
	}

	if i.NetworkRoles == nil {
		i.NetworkRoles = []string{}
	}

	return
}

func (i *Instance) Json() {
	switch i.State {
	case Running:
		switch i.VmState {
		case vm.Starting:
			i.Status = "Starting"
			break
		case vm.Running:
			i.Status = "Running"
			break
		case vm.Stopped:
			i.Status = "Starting"
			break
		case vm.Failed:
			i.Status = "Starting"
			break
		case vm.Updating:
			i.Status = "Updating"
			break
		case vm.ProvisioningDisk:
			i.Status = "Provisioning Disk"
			break
		case "":
			i.Status = "Provisioning"
			break
		}
		break
	case Stopped:
		switch i.VmState {
		case vm.Starting:
			i.Status = "Stopping"
			break
		case vm.Running:
			i.Status = "Stopping"
			break
		case vm.Stopped:
			i.Status = "Stopped"
			break
		case vm.Failed:
			i.Status = "Stopped"
			break
		case vm.Updating:
			i.Status = "Updating"
			break
		case vm.ProvisioningDisk:
			i.Status = "Provisioning Disk"
			break
		case "":
			i.Status = "Provisioning"
			break
		}
		break
	case Updating:
		i.Status = "Updating"
		break
	case Deleting:
		i.Status = "Deleting"
		break
	case Snapshot:
		i.Status = "Snapshotting"
		break
	}
}

func (i *Instance) Commit(db *database.Database) (err error) {
	coll := db.Instances()

	err = coll.Commit(i.Id, i)
	if err != nil {
		return
	}

	return
}

func (i *Instance) CommitFields(db *database.Database, fields set.Set) (
	err error) {

	coll := db.Instances()

	err = coll.CommitFields(i.Id, i, fields)
	if err != nil {
		return
	}

	return
}

func (i *Instance) Insert(db *database.Database) (err error) {
	coll := db.Instances()

	if i.Id != "" {
		err = &errortypes.DatabaseError{
			errors.New("instance: Instance already exists"),
		}
		return
	}

	err = coll.Insert(i)
	if err != nil {
		err = database.ParseError(err)
		return
	}

	return
}

func (i *Instance) GetVm(disks []*disk.Disk) (virt *vm.VirtualMachine) {
	virt = &vm.VirtualMachine{
		Id:         i.Id,
		Image:      i.Image,
		Processors: i.Processors,
		Memory:     i.Memory,
		Disks:      []*vm.Disk{},
		NetworkAdapters: []*vm.NetworkAdapter{
			&vm.NetworkAdapter{
				MacAddress:    vm.GetMacAddr(i.Id),
				HostInterface: bridge.BridgeName,
			},
		},
	}

	if disks != nil {
		for _, dsk := range disks {
			index, err := strconv.Atoi(dsk.Index)
			if err != nil {
				continue
			}

			virt.Disks = append(virt.Disks, &vm.Disk{
				Index: index,
				Path:  vm.GetDiskPath(dsk.Id),
			})
		}
	}

	return
}

func (i *Instance) Changed(virt *vm.VirtualMachine) bool {
	return i.Memory != virt.Memory || i.Processors != virt.Processors
}
