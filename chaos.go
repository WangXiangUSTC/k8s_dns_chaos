package kubernetes

import (
	"context"
	"math/rand"
	"net"

	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	api "k8s.io/api/core/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (k Kubernetes) chaosDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg, state request.Request) (int, error) {
	answers := []dns.RR{}
	qname := state.Name()

	// TODO: support more type
	switch state.QType() {
	case dns.TypeA:
		ips := []net.IP{getRandomIPv4()}
		log.Infof("dns.TypeA %v", ips)
		answers = a(qname, 10, ips)
	case dns.TypeAAAA:
		ips := []net.IP{net.IP{0x20, 0x1, 0xd, 0xb8, 0, 0, 0, 0, 0, 0, 0x1, 0x23, 0, 0x12, 0, 0x1}}
		log.Infof("dns.TypeAAAA %v", ips)
		answers = aaaa(qname, 10, ips)
	}

	if len(answers) == 0 {
		return dns.RcodeServerFailure, nil
	}

	log.Infof("answers %v", answers)

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	m.Answer = answers

	w.WriteMsg(m)
	return dns.RcodeSuccess, nil

}

func getRandomIPv4() net.IP {
	nums := make([]byte, 0, 4)

	for i := 0; i < 4; i++ {
		nums = append(nums, byte(rand.Intn(255)))
	}

	return net.IPv4(nums[0], nums[1], nums[2], nums[3])
}

// a takes a slice of net.IPs and returns a slice of A RRs.
func a(zone string, ttl uint32, ips []net.IP) []dns.RR {
	answers := make([]dns.RR, len(ips))
	for i, ip := range ips {
		r := new(dns.A)
		r.Hdr = dns.RR_Header{Name: zone, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl}
		r.A = ip
		answers[i] = r
	}
	return answers
}

// aaaa takes a slice of net.IPs and returns a slice of AAAA RRs.
func aaaa(zone string, ttl uint32, ips []net.IP) []dns.RR {
	answers := make([]dns.RR, len(ips))
	for i, ip := range ips {
		r := new(dns.AAAA)
		r.Hdr = dns.RR_Header{Name: zone, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: ttl}
		r.AAAA = ip
		answers[i] = r
	}
	return answers
}

func (k Kubernetes) getChaosMode(pod *api.Pod) string {
	k.RLock()
	defer k.RUnlock()

	if pod == nil {
		return ""
	}

	if _, ok := k.podChaosMap[pod.Namespace]; ok {
		return k.podChaosMap[pod.Namespace][pod.Name]
	}

	return ""
}

func (k Kubernetes) getChaosPod() ([]api.Pod, error) {
	k.RLock()
	defer k.RUnlock()

	pods := make([]api.Pod, 0, 10)
	for namespace := range k.podChaosMap {
		podList, err := k.Client.Pods(namespace).List(context.Background(), meta.ListOptions{})
		if err != nil {
			return nil, err
		}
		for _, pod := range podList.Items {
			pods = append(pods, pod)
		}
	}

	return pods, nil
}
