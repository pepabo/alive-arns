package arn

import (
	"context"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	taggingapi "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"golang.org/x/sync/errgroup"
)

var maxItem int32 = 1000

type Arn struct {
	arn    string
	region *string
}

func (a *Arn) String() string {
	return a.arn
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
	mu     sync.Mutex
	once   bool
	region string
}

func New() *Collector {
	return &Collector{}
}

func (c *Collector) CollectArns(ctx context.Context, cfg aws.Config) (Arns, error) {
	c.region = cfg.Region
	arns, err := collectArnsUsingTaggingApi(ctx, cfg)
	if err != nil {
		return nil, err
	}
	arns2, err := c.collectArns(ctx, cfg)
	if err != nil {
		return nil, err
	}
	arns = append(arns, arns2...)
	return arns.Unique(), nil
}

func (c *Collector) collectArns(ctx context.Context, cfg aws.Config) (Arns, error) {
	arns := Arns{}
	// clients := []interface{}
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
					var marker *string
					for {
						var (
							a   Arns
							err error
						)
						a, marker, err = c.callListMethod(ctx, client, m, marker)
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

func (c *Collector) callListMethod(ctx context.Context, client interface{}, m reflect.Method, marker *string) (Arns, *string, error) {
	if !strings.HasPrefix(m.Name, "List") {
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

				// Pagination
				f := input.Elem().FieldByName("Marker")
				if f.IsValid() {
					if f.CanSet() && marker != nil {
						f.Set(reflect.ValueOf(marker))
					}
				}

				{
					f := input.Elem().FieldByName("MaxItems")
					if f.IsValid() {
						if f.CanSet() {
							f.Set(reflect.ValueOf(&maxItem))
						}
					}
				}

				argv[i] = reflect.ValueOf(input.Interface())
			}
		}
	}
	result := m.Func.Call(argv)
	if result[0].IsNil() {
		return nil, nil, nil
	}
	r := result[0].Elem()
	for i := 0; i < r.NumField(); i++ {
		if r.Field(i).Kind() == reflect.Slice {
			for j := 0; j < r.Field(i).Len(); j++ {
				rr := r.Field(i).Index(j)
				arn, ok := getFieldString(rr, "Arn")
				if ok {
					if strings.Contains(arn, c.region) {
						arns = append(arns, &Arn{
							region: &c.region,
							arn:    arn,
						})
					} else {
						arns = append(arns, &Arn{
							region: nil,
							arn:    arn,
						})
					}
				}
			}
		}
	}
	var next *string
	mrk, ok := getFieldString(r, "Marker")
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
	region := cfg.Region
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
			arn := *r.ResourceARN
			if strings.Contains(arn, region) {
				arns = append(arns, &Arn{
					region: &region,
					arn:    arn,
				})
			} else {
				arns = append(arns, &Arn{
					region: nil,
					arn:    arn,
				})
			}
		}
		if o.PaginationToken == nil || *o.PaginationToken == "" {
			break
		}
		token = o.PaginationToken
	}
	return arns, nil
}
