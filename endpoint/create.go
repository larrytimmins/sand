package endpoint

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/Scalingo/sand/api/params"
	"github.com/Scalingo/sand/api/types"
	"github.com/Scalingo/sand/ipallocator"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
)

func (r *repository) Create(ctx context.Context, n types.Network, params params.EndpointCreate) (types.Endpoint, error) {
	var endpoint types.Endpoint

	allocator := ipallocator.New(r.config, r.store, n.ID, ipallocator.WithIPRange(n.IPRange))

	ipOpts := ipallocator.AllocateIPOpts{
		Address: params.IPv4Address,
	}
	ip, mask, err := allocator.AllocateIP(ctx, ipOpts)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to allocate IP for endpoint")
	}

	macAddress := ipv4ToMac(ip)
	if params.MacAddress != "" {
		macAddress = params.MacAddress
	}

	endpoint = types.Endpoint{
		ID:            uuid.NewRandom().String(),
		Hostname:      r.config.PublicHostname,
		HostIP:        r.config.PublicIP,
		NetworkID:     n.ID,
		CreatedAt:     time.Now(),
		TargetVethIP:  fmt.Sprintf("%s/%d", ip.String(), mask),
		TargetVethMAC: macAddress,
	}

	err = r.store.Set(ctx, endpoint.StorageKey(), &endpoint)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to save endpoint %s in store", endpoint)
	}

	err = r.store.Set(ctx, endpoint.NetworkStorageKey(), &endpoint)
	if err != nil {
		return endpoint, errors.Wrapf(err, "fail to save endpoint %s in store network", endpoint)
	}

	if params.Activate {
		endpoint, err = r.Activate(ctx, n, endpoint, params.ActivateParams)
		if err != nil {
			return endpoint, errors.Wrapf(err, "fail to ensure endpoint")
		}
	}

	return endpoint, nil
}

func ipv4ToMac(ip net.IP) string {
	ip = ip.To4()
	return fmt.Sprintf("02:42:%02x:%02x:%02x:%02x", ip[0], ip[1], ip[2], ip[3])
}