package main

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"

	"github.com/graph-gophers/graphql-go"
	"github.com/graph-gophers/graphql-go/relay"
)

//go:embed index.html
var page []byte

//go:embed schema.graphql
var sdl string

type entitiesFunc func(args struct{ Representations []Any }) ([]*Entity, error)

type resolver struct {
	depProducts map[string]DeprecatedProduct
	products    map[graphql.ID]Product

	Entities entitiesFunc   `graphql:"_entities"`
	Service  func() Service `graphql:"_service"`
}

func (r *resolver) Product(args struct{ ID graphql.ID }) (*Product, error) {
	p, ok := r.products[args.ID]
	if !ok {
		return nil, fmt.Errorf("product not found")
	}
	return &p, nil
}

type Product struct {
	ID         graphql.ID
	SKU        *string
	Package    *string
	Variation  *ProductVariation
	Dimensions *ProductDimension
	CreatedBy  *User
	Notes      *string
	Research   []ProductResearch
}

type ProductVariation struct {
	ID graphql.ID
}

type ProductDimension struct {
	Size   *string
	Weight *float64
	Unit   *string
}

type ProductResearch struct {
	Study   CaseStudy
	Outcome *string
}

type CaseStudy struct {
	CaseNumber  graphql.ID
	Description *string
}

type DeprecatedProductArgs struct {
	SKU     string
	Package string
}

func (r *resolver) DeprecatedProduct(args *DeprecatedProductArgs) (*DeprecatedProduct, error) {
	if args == nil {
		return nil, fmt.Errorf("args required")
	}
	key := args.SKU + "-" + args.Package
	p, ok := r.depProducts[key]
	if !ok {
		return nil, fmt.Errorf("product not found")
	}
	return &p, nil
}

type DeprecatedProduct struct {
	SKU       string
	Package   string
	Reason    *string
	CreatedBy *User
}

type User struct {
	AverageProductsCreatedPerYear *int32
	Email                         graphql.ID
	Name                          *string
	TotalProductsCreated          *int32
	YearsOfEmployment             int32
}

type Inventory struct {
	ID                 graphql.ID
	DeprecatedProducts []DeprecatedProduct
}

func entities(depProds map[string]DeprecatedProduct, invs map[graphql.ID]Inventory, products map[graphql.ID]Product, researches map[graphql.ID]ProductResearch, users map[graphql.ID]User) entitiesFunc {
	pkgKey := func(sku, pkg string) string { return sku + "-" + pkg }
	productsByPkg := map[string]graphql.ID{}
	for _, p := range products {
		if p.SKU != nil && p.Package != nil {
			productsByPkg[pkgKey(*p.SKU, *p.Package)] = p.ID
		}
	}

	variationKey := func(sku string, variationID graphql.ID) string { return sku + "-" + string(variationID) }
	productsByVariation := map[string]graphql.ID{}
	for _, p := range products {
		if p.SKU != nil && p.Variation != nil {
			productsByVariation[variationKey(*p.SKU, p.Variation.ID)] = p.ID
		}
	}

	return func(args struct{ Representations []Any }) ([]*Entity, error) {
		var res []*Entity
		for _, rep := range args.Representations {
			switch rep.TypeName {
			case "DeprecatedProduct":
				var prod *DeprecatedProduct
				key := rep.Key.(DepProdKey)
				p, ok := depProds[pkgKey(key.SKU, key.Package)]
				if ok {
					prod = &p
				}
				res = append(res, &Entity{entity: prod})
			case "Inventory":
				var inv *Inventory
				key := rep.Key.(InvKey)
				i, ok := invs[key.ID]
				if ok {
					inv = &i
				}
				res = append(res, &Entity{entity: inv})
			case "Product":
				var prod *Product
				key := rep.Key.(ProdKey)
				if key.ID != nil {
					p, ok := products[*key.ID]
					if ok {
						prod = &p
					}
				} else if key.SKU != nil { // next two checks require SKU
					if key.Package != nil {
						id, ok := productsByPkg[pkgKey(*key.SKU, *key.Package)]
						if ok {
							p := products[id]
							prod = &p
						}
					} else if key.Variation != nil {
						id, ok := productsByVariation[variationKey(*key.SKU, key.Variation.ID)]
						if ok {
							p := products[id]
							prod = &p
						}
					}
				}
				res = append(res, &Entity{entity: prod})
			case "ProductResearch":
				var pr *ProductResearch
				key := rep.Key.(ProdResKey)
				r, ok := researches[key.Study.ID]
				if ok {
					pr = &r
				}
				res = append(res, &Entity{entity: pr})
			case "User":
				var usr *User
				key := rep.Key.(UserKey)
				u, ok := users[key.Email]
				if ok {
					usr = &u
				}
				res = append(res, &Entity{entity: usr})
			default:
				return nil, fmt.Errorf("unexpected representation type %q", rep.TypeName)
			}
		}

		return res, nil
	}
}

type Entity struct {
	entity interface{}
}

func (e *Entity) ToProduct() (*Product, bool) {
	p, ok := e.entity.(*Product)
	return p, ok
}

func (e *Entity) ToDeprecatedProduct() (*DeprecatedProduct, bool) {
	p, ok := e.entity.(*DeprecatedProduct)
	return p, ok
}

func (e *Entity) ToProductResearch() (*ProductResearch, bool) {
	p, ok := e.entity.(*ProductResearch)
	return p, ok
}

func (e *Entity) ToUser() (*User, bool) {
	u, ok := e.entity.(*User)
	return u, ok
}

func (e *Entity) ToInventory() (*Inventory, bool) {
	i, ok := e.entity.(*Inventory)
	return i, ok
}

func service(s string) func() Service {
	return func() Service {
		return Service{SDL: s}
	}
}

type Service struct {
	SDL string
}

func populateResolver(sdl string) *resolver {
	defaultUser := &User{
		Email:                         graphql.ID("support@apollographql.com"),
		Name:                          strptr("Jane Smith"),
		TotalProductsCreated:          intptr(1337),
		AverageProductsCreatedPerYear: intptr(134),
		YearsOfEmployment:             10,
	}
	users := map[graphql.ID]User{
		defaultUser.Email: *defaultUser,
	}

	prodResearch1 := ProductResearch{
		Study: CaseStudy{
			CaseNumber:  "1234",
			Description: strptr("Federation Study"),
		},
	}
	prodResearch2 := ProductResearch{
		Study: CaseStudy{
			CaseNumber:  "1235",
			Description: strptr("Studio Study"),
		},
	}
	researches := map[graphql.ID]ProductResearch{
		prodResearch1.Study.CaseNumber: prodResearch1,
		prodResearch2.Study.CaseNumber: prodResearch2,
	}

	dim := ProductDimension{
		Size:   strptr("small"),
		Weight: floatptr(1),
		Unit:   strptr("kg"),
	}
	prod1 := Product{
		ID:         "apollo-federation",
		SKU:        strptr("federation"),
		Dimensions: &dim,
		CreatedBy:  defaultUser,
		Package:    strptr("@apollo/federation"),
		Variation:  &ProductVariation{ID: "OSS"},
		Research:   []ProductResearch{prodResearch1},
	}
	prod2 := Product{
		ID:         "apollo-studio",
		SKU:        strptr("studio"),
		Dimensions: &dim,
		CreatedBy:  defaultUser,
		Variation:  &ProductVariation{ID: "platform"},
		Research:   []ProductResearch{prodResearch2},
	}
	products := map[graphql.ID]Product{
		prod1.ID: prod1,
		prod2.ID: prod2,
	}

	depProduct1 := DeprecatedProduct{
		SKU:       "apollo-federation-v1",
		Package:   "@apollo/federation-v1",
		Reason:    strptr("Migrate to Federation V2"),
		CreatedBy: defaultUser,
	}
	depProducts := map[string]DeprecatedProduct{
		depProduct1.SKU + "-" + depProduct1.Package: depProduct1,
	}

	inv := Inventory{
		ID:                 graphql.ID("apollo-oss"),
		DeprecatedProducts: []DeprecatedProduct{depProduct1},
	}
	invs := map[graphql.ID]Inventory{
		inv.ID: inv,
	}

	return &resolver{
		depProducts: depProducts,
		products:    products,
		Entities:    entities(depProducts, invs, products, researches, users),
		Service:     service(sdl),
	}
}

func intptr(i int32) *int32 {
	return &i
}

func strptr(s string) *string {
	return &s
}

func floatptr(f float64) *float64 {
	return &f
}

func main() {
	r := populateResolver(sdl)
	opts := []graphql.SchemaOpt{graphql.UseStringDescriptions(), graphql.UseFieldResolvers()}
	schema := graphql.MustParseSchema(sdl, r, opts...)
	http.HandleFunc("/graphiql", func(w http.ResponseWriter, r *http.Request) { w.Write(page) })
	http.Handle("/", &relay.Handler{Schema: schema})

	log.Fatal(http.ListenAndServe(":4001", nil))
}
