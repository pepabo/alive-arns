package arn

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	taggingapi "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/securityhub"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

var maxItem int32 = 100

type Arn struct {
	Arn       string `json:"arn"`
	Partition string `json:"partition"`
	Service   string `json:"service"`
	// omitempty is not set because the output is not easy to use
	Region       string `json:"region"`
	AccountID    string `json:"account_id"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
}

func New(arn string) (*Arn, error) {
	splitted := strings.Split(arn, ":")
	if len(splitted) != 6 && len(splitted) != 7 {
		return nil, fmt.Errorf("invalid arn format: %s", arn)
	}
	a := &Arn{
		Arn:       arn,
		Partition: splitted[1],
		Service:   splitted[2],
		Region:    splitted[3],
		AccountID: splitted[4],
	}
	switch len(splitted) {
	case 6:
		splitted2 := strings.SplitN(splitted[5], "/", 2)
		switch len(splitted) {
		case 2:
			a.ResourceType = splitted2[0]
			a.ResourceID = splitted2[1]
		case 1:
			a.ResourceID = splitted2[0]
		}
	case 7:
		a.ResourceType = splitted[5]
		a.ResourceID = splitted[6]
	}
	return a, nil
}

func (a *Arn) String() string {
	return a.Arn
}

type Arns []*Arn

func (arns Arns) Unique() Arns {
	u := Arns{}
	m := map[string]struct{}{}
	for _, arn := range arns {
		if _, ok := m[arn.String()]; ok {
			continue
		}
		u = append(u, arn)
		m[arn.String()] = struct{}{}
	}
	return u
}

func (arns Arns) Sort() Arns {
	sort.SliceStable(arns, func(i, j int) bool {
		return arns[i].String() < arns[j].String()
	})
	return arns
}

type Collector struct {
	mu   sync.Mutex
	once bool
}

func NewCollector() *Collector {
	return &Collector{}
}

func (c *Collector) CollectArns(ctx context.Context, cfg aws.Config) (Arns, error) {
	arns, err := collectArnsUsingTaggingApi(ctx, cfg)
	if err != nil {
		return nil, err
	}
	arns2, err := c.collectArnsUsingReflect(ctx, cfg)
	if err != nil {
		return nil, err
	}
	arns = append(arns, arns2...)
	arns3, err := c.collectArnsSecurityhubExtras(ctx, cfg)
	if err != nil {
		return nil, err
	}
	arns = append(arns, arns3...)

	return arns.Unique(), nil
}

func (c *Collector) collectArnsUsingReflect(ctx context.Context, cfg aws.Config) (Arns, error) {
	arns := Arns{}
	clients := []interface{}{
		securityhub.NewFromConfig(cfg),
	}
	clientsOnce := []interface{}{
		iam.NewFromConfig(cfg),
	}

	eg := new(errgroup.Group)

	callMethods := func(client interface{}) {
		cType := reflect.TypeOf(client)
		for i := 0; i < cType.NumMethod(); i++ {
			func(client interface{}, i int) {
				eg.Go(func() error {
					m := cType.Method(i)
					if containsPrefix(methodPrefixIgnores, m.Name) || containsSuffix(methodSuffixIgnores, m.Name) {
						return nil
					}

					key := fmt.Sprintf("%s.%s", cType.Elem().String(), m.Name)
					if contains(ignores, key) || contains(extras, key) {
						return nil
					}
					var marker *string
					for {
						var (
							a   Arns
							err error
						)
						a, marker, err = c.callListOrDescribeMethod(ctx, client, m, marker)
						if err != nil {
							return err
						}
						c.mu.Lock()
						arns = append(arns, a...)
						c.mu.Unlock()
						if marker == nil {
							break
						}
					}
					return nil
				})
			}(client, i)
		}
	}

	for _, client := range clients {
		callMethods(client)
	}
	if !c.once {
		for _, client := range clientsOnce {
			callMethods(client)
		}
		c.once = true
	}
	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return arns, nil
}

func (c *Collector) callListOrDescribeMethod(ctx context.Context, client interface{}, m reflect.Method, marker *string) (Arns, *string, error) {
	if !strings.HasPrefix(m.Name, "List") && !strings.HasPrefix(m.Name, "Describe") {
		return nil, nil, nil
	}
	arns := Arns{}
	t := m.Func.Type()
	argv := make([]reflect.Value, t.NumIn()-1)
	for i := range argv {
		switch t.In(i) {
		case reflect.TypeOf(client):
			argv[i] = reflect.ValueOf(client)
		case reflect.TypeOf((*context.Context)(nil)).Elem():
			argv[i] = reflect.ValueOf(ctx)
		default:
			if t.In(i).Kind() == reflect.Ptr && strings.HasSuffix(t.In(i).Elem().Name(), "Input") {
				// *Input
				input := reflect.New(t.In(i).Elem())

				// Pagination Token
				for _, tn := range []string{"Marker", "NextToken"} {
					f := input.Elem().FieldByName(tn)
					if f.IsValid() {
						if f.CanSet() && marker != nil {
							f.Set(reflect.ValueOf(marker))
						}
					}
				}

				// Max
				for _, m := range []string{"MaxItems"} {
					f := input.Elem().FieldByName(m)
					if f.IsValid() {
						if f.CanSet() {
							f.Set(reflect.ValueOf(&maxItem))
						}
					}
				}
				for _, m := range []string{"MaxResults"} {
					f := input.Elem().FieldByName(m)
					if f.IsValid() {
						if f.CanSet() {
							f.Set(reflect.ValueOf(maxItem))
						}
					}
				}

				argv[i] = reflect.ValueOf(input.Interface())
			}
		}
	}

	result := m.Func.Call(argv)
	if len(result) > 1 && !result[1].IsNil() {
		return nil, nil, result[1].Interface().(error)
	}
	if result[0].IsNil() {
		return nil, nil, nil
	}
	r := result[0].Elem()
	arn, ok := getFieldString(r, "Arn")
	if ok {
		arns = appendARN(arns, arn)
	}

	for i := 0; i < r.NumField(); i++ {
		if r.Field(i).Kind() == reflect.Slice {
			for j := 0; j < r.Field(i).Len(); j++ {
				rr := r.Field(i).Index(j)
				if rr.Kind() == reflect.String {
					// []string
					arns = appendARN(arns, rr.String())
					continue
				}
				// []struct
				arn, ok := getFieldString(rr, "Arn")
				if ok {
					arns = appendARN(arns, arn)
				}
			}
		}
	}
	var next *string
	mrk, ok := getFieldString(r, "Marker")
	if !ok {
		mrk, ok = getFieldString(r, "NextToken")
	}
	if ok && mrk != "" {
		next = &mrk
	}
	return arns, next, nil
}

func getFieldString(s reflect.Value, contains string) (string, bool) {
	for k := 0; k < s.NumField(); k++ {
		if strings.Contains(s.Type().Field(k).Name, contains) {
			if s.Field(k).IsNil() {
				return "", false
			}
			if s.Field(k).Kind() == reflect.Ptr {
				return s.Field(k).Elem().String(), true
			} else {
				return s.Field(k).String(), true
			}
		}
	}
	return "", false
}

func collectArnsUsingTaggingApi(ctx context.Context, cfg aws.Config) (Arns, error) {
	var token *string
	arns := Arns{}
	c := taggingapi.NewFromConfig(cfg)

	for {
		o, err := c.GetResources(ctx, &taggingapi.GetResourcesInput{
			PaginationToken: token,
		})
		if err != nil {
			return nil, err
		}
		for _, r := range o.ResourceTagMappingList {
			arns = appendARN(arns, *r.ResourceARN)
		}
		if o.PaginationToken == nil || *o.PaginationToken == "" {
			break
		}
		token = o.PaginationToken
	}

	return arns.Unique(), nil
}

func appendARN(arns Arns, arn string) Arns {
	a, err := New(arn)
	if err == nil {
		arns = append(arns, a)
	} else {
		log.Debug().Err(err).Str("arn", arn)
	}
	return arns
}

func contains(s []string, e string) bool {
	for _, v := range s {
		if e == v {
			return true
		}
	}
	return false
}

func containsPrefix(s []string, e string) bool {
	for _, v := range s {
		if strings.HasPrefix(e, v) {
			return true
		}
	}
	return false
}

func containsSuffix(s []string, e string) bool {
	for _, v := range s {
		if strings.HasSuffix(e, v) {
			return true
		}
	}
	return false
}
