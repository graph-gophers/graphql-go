package graphql_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/graph-gophers/graphql-go"
)

// In this example we demonstrate a 3-level hierarchy (Category -> Products -> Reviews)
// and show how to prefetch nested data (products & reviews) in a single pass using
// the selected field/argument inspection helpers.
// The type names are prefixed with "pf" to avoid clashing with other examples in this package.
type pfCategory struct {
	ID   string
	Name string
}
type pfProduct struct {
	ID         string
	CategoryID string
	Name       string
	Price      int
}
type pfReview struct {
	ID        string
	ProductID string
	Body      string
	Stars     int32
}

var (
	pfCategories = []pfCategory{{"C1", "Electronics"}}
	pfProducts   = []pfProduct{
		{"P01", "C1", "Adapter", 15},
		{"P02", "C1", "Battery", 25},
		{"P03", "C1", "Cable", 5},
		{"P04", "C1", "Dock", 45},
		{"P05", "C1", "Earbuds", 55},
		{"P06", "C1", "Fan", 35},
		{"P07", "C1", "Gamepad", 65},
		{"P08", "C1", "Hub", 40},
		{"P09", "C1", "Indicator", 12},
		{"P10", "C1", "Joystick", 70},
		{"P11", "C1", "Keyboard", 80},
		{"P12", "C1", "Light", 8},
		{"P13", "C1", "Microphone", 120},
	}
	pfReviews = []pfReview{
		{"R01", "P05", "Great sound", 5},
		{"R02", "P05", "Decent", 4},
		{"R03", "P05", "Could be louder", 3},
		{"R04", "P05", "Nice fit", 5},
		{"R05", "P05", "Battery ok", 4},
		{"R06", "P05", "Color faded", 2},
		{"R07", "P05", "Value for money", 5},
		{"R08", "P11", "Fast typing", 5},
		{"R09", "P11", "Loud keys", 3},
		{"R10", "P02", "Holds charge", 4},
		{"R11", "P02", "Gets warm", 2},
	}
)

// SDL describing the hierarchy with pagination & ordering arguments.
const prefetchSDL = `
schema { query: Query }

enum ProductOrder {
	NAME
	PRICE
}

type Query {
	category(id: ID!): Category
}

type Category {
	id: ID!
	name: String!
	products(after: ID, first: Int, orderBy: ProductOrder): [Product!]!
}

type Product {
	id: ID!
	name: String!
	price: Int!
	reviews(last: Int = 5): [Review!]!
}

type Review {
	id: ID!
	body: String!
	stars: Int!
}
`

type pfRoot struct{}

// ProductOrder represented as plain string for simplicity in this example.
type ProductOrder string

const (
	ProductOrderName  ProductOrder = "NAME"
	ProductOrderPrice ProductOrder = "PRICE"
)

func (r *pfRoot) Category(ctx context.Context, args struct{ ID graphql.ID }) *pfCategoryResolver {
	var cat *pfCategory
	for i := range pfCategories {
		if pfCategories[i].ID == string(args.ID) {
			cat = &pfCategories[i]
			break
		}
	}
	if cat == nil {
		return nil
	}

	cr := &pfCategoryResolver{c: cat}

	// Exit early if "products" field wasn't requested
	if !graphql.HasSelectedField(ctx, "products") {
		return cr
	}

	// Prefetch products for this category
	// Decode any arguments provided to the "products" field
	// and apply them during prefetching.
	var prodArgs struct {
		After   graphql.ID
		First   *int32
		OrderBy *string
	}
	_, _ = graphql.DecodeSelectedFieldArgs(ctx, "products", &prodArgs)
	firstVal := int32(10)
	if prodArgs.First != nil && *prodArgs.First > 0 {
		firstVal = *prodArgs.First
	}
	orderVal := ProductOrderName
	if prodArgs.OrderBy != nil && *prodArgs.OrderBy != "" {
		orderVal = ProductOrder(*prodArgs.OrderBy)
	}
	filtered := make([]pfProduct, 0, 16)
	for _, p := range pfProducts {
		if p.CategoryID == cat.ID {
			filtered = append(filtered, p)
		}
	}
	switch orderVal {
	case ProductOrderPrice:
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].Price < filtered[j].Price })
	default:
		sort.Slice(filtered, func(i, j int) bool { return filtered[i].Name < filtered[j].Name })
	}
	var start int
	if prodArgs.After != "" {
		for i, p := range filtered {
			if p.ID == string(prodArgs.After) {
				start = i + 1
				break
			}
		}
		if start > len(filtered) {
			start = len(filtered)
		}
	}
	end := start + int(firstVal)
	if end > len(filtered) {
		end = len(filtered)
	}
	slice := filtered[start:end]
	cr.prefetchedProducts = make([]*pfProduct, len(slice))
	for i := range slice {
		prod := slice[i]
		cr.prefetchedProducts[i] = &prod
	}

	// Exit early if "reviews" sub-field wasn't requested for the products
	if !graphql.HasSelectedField(ctx, "products.reviews") {
		return cr
	}

	// Prefetch reviews for all products in this category
	// Decode any arguments provided to the "reviews" field
	// and apply them during prefetching.
	var reviewArgs struct{ Last int32 }
	_, _ = graphql.DecodeSelectedFieldArgs(ctx, "products.reviews", &reviewArgs)
	var lastVal int32
	if reviewArgs.Last > 0 {
		lastVal = reviewArgs.Last
	}
	take := int(lastVal)
	cr.reviewsByProduct = make(map[string][]*pfReview)
	productSet := map[string]struct{}{}
	for _, p := range cr.prefetchedProducts {
		productSet[p.ID] = struct{}{}
	}
	for i := range pfReviews {
		rv := pfReviews[i]
		if _, ok := productSet[rv.ProductID]; !ok {
			continue
		}
		arr := cr.reviewsByProduct[rv.ProductID]
		arr = append(arr, &rv)
		if take > 0 && len(arr) > take {
			arr = arr[len(arr)-take:]
		}
		cr.reviewsByProduct[rv.ProductID] = arr
	}
	return cr
}

type pfCategoryResolver struct {
	c                  *pfCategory
	prefetchedProducts []*pfProduct
	reviewsByProduct   map[string][]*pfReview
}

func (c *pfCategoryResolver) ID() graphql.ID { return graphql.ID(c.c.ID) }
func (c *pfCategoryResolver) Name() string   { return c.c.Name }

type pfProductArgs struct {
	After   *graphql.ID
	First   *int32
	OrderBy *string
}

func (c *pfCategoryResolver) Products(ctx context.Context, args pfProductArgs) ([]*pfProductResolver, error) {
	out := make([]*pfProductResolver, len(c.prefetchedProducts))
	for i, p := range c.prefetchedProducts {
		out[i] = &pfProductResolver{parent: c, p: p}
	}
	return out, nil
}

type pfProductResolver struct {
	parent *pfCategoryResolver
	p      *pfProduct
}

func (p *pfProductResolver) ID() graphql.ID { return graphql.ID(p.p.ID) }
func (p *pfProductResolver) Name() string   { return p.p.Name }
func (p *pfProductResolver) Price() int32   { return int32(p.p.Price) }
func (p *pfProductResolver) Reviews(ctx context.Context, args struct{ Last int32 }) ([]*pfReviewResolver, error) {
	rs := p.parent.reviewsByProduct[p.p.ID]
	out := make([]*pfReviewResolver, len(rs))
	for i, r := range rs {
		out[i] = &pfReviewResolver{r: r}
	}
	return out, nil
}

type pfReviewResolver struct{ r *pfReview }

func (r *pfReviewResolver) ID() graphql.ID { return graphql.ID(r.r.ID) }
func (r *pfReviewResolver) Body() string   { return r.r.Body }
func (r *pfReviewResolver) Stars() int32   { return r.r.Stars }

// ExamplePrefetchData demonstrates data prefetching for a 3-level hierarchy depending on the requested fields.
func Example_prefetchData() {
	schema := graphql.MustParseSchema(prefetchSDL, &pfRoot{})

	// Query 1: order products by NAME, starting after P02, first 5, with default last 5 reviews.
	q1 := `{
  category(id:"C1") {
    id
    name
    products(after:"P02", first:5, orderBy: NAME) {
      id
      name
      price
      reviews {
        id
        stars
      }
    }
  }
}`

	// Query 2: order by PRICE, no cursor (after), first 4 products only.
	q2 := `{
  category(id:"C1") {
    products(first:4, orderBy: PRICE) {
      id
      name
      price
    }
  }
}`

	fmt.Println("Order by NAME result:")
	res1 := schema.Exec(context.Background(), q1, "", nil)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(res1)

	fmt.Println("Order by PRICE result:")
	res2 := schema.Exec(context.Background(), q2, "", nil)
	enc = json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(res2)

	// output:
	// Order by NAME result:
	// {
	//   "data": {
	//     "category": {
	//       "id": "C1",
	//       "name": "Electronics",
	//       "products": [
	//         {
	//           "id": "P03",
	//           "name": "Cable",
	//           "price": 5,
	//           "reviews": []
	//         },
	//         {
	//           "id": "P04",
	//           "name": "Dock",
	//           "price": 45,
	//           "reviews": []
	//         },
	//         {
	//           "id": "P05",
	//           "name": "Earbuds",
	//           "price": 55,
	//           "reviews": [
	//             {
	//               "id": "R01",
	//               "stars": 5
	//             },
	//             {
	//               "id": "R02",
	//               "stars": 4
	//             },
	//             {
	//               "id": "R03",
	//               "stars": 3
	//             },
	//             {
	//               "id": "R04",
	//               "stars": 5
	//             },
	//             {
	//               "id": "R05",
	//               "stars": 4
	//             },
	//             {
	//               "id": "R06",
	//               "stars": 2
	//             },
	//             {
	//               "id": "R07",
	//               "stars": 5
	//             }
	//           ]
	//         },
	//         {
	//           "id": "P06",
	//           "name": "Fan",
	//           "price": 35,
	//           "reviews": []
	//         },
	//         {
	//           "id": "P07",
	//           "name": "Gamepad",
	//           "price": 65,
	//           "reviews": []
	//         }
	//       ]
	//     }
	//   }
	// }
	// Order by PRICE result:
	// {
	//   "data": {
	//     "category": {
	//       "products": [
	//         {
	//           "id": "P03",
	//           "name": "Cable",
	//           "price": 5
	//         },
	//         {
	//           "id": "P12",
	//           "name": "Light",
	//           "price": 8
	//         },
	//         {
	//           "id": "P09",
	//           "name": "Indicator",
	//           "price": 12
	//         },
	//         {
	//           "id": "P01",
	//           "name": "Adapter",
	//           "price": 15
	//         }
	//       ]
	//     }
	//   }
	// }
}
