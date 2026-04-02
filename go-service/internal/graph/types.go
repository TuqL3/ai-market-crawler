package graph

import "github.com/graphql-go/graphql"

var RawProblemType = graphql.NewObject(graphql.ObjectConfig{
	Name: "RawProblem",
	Fields: graphql.Fields{
		"id":            &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
		"source":        &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"sourceId":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"url":           &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"title":         &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"body":          &graphql.Field{Type: graphql.String},
		"tags":          &graphql.Field{Type: graphql.NewList(graphql.String)},
		"score":         &graphql.Field{Type: graphql.Int},
		"answerCount":   &graphql.Field{Type: graphql.Int},
		"commentCount":  &graphql.Field{Type: graphql.Int},
		"sourceCreated": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"crawledAt":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
	},
})

var ClassifiedProblemType = graphql.NewObject(graphql.ObjectConfig{
	Name: "ClassifiedProblem",
	Fields: graphql.Fields{
		"id":            &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
		"rawProblemId":  &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
		"category":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"subcategories": &graphql.Field{Type: graphql.NewList(graphql.String)},
		"confidence":    &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
		"classifiedAt":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
	},
})

var ProblemClusterType = graphql.NewObject(graphql.ObjectConfig{
	Name: "ProblemCluster",
	Fields: graphql.Fields{
		"id":              &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
		"label":           &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"summary":         &graphql.Field{Type: graphql.String},
		"keyThemes":       &graphql.Field{Type: graphql.NewList(graphql.String)},
		"commonSolutions": &graphql.Field{Type: graphql.NewList(graphql.String)},
		"cohesionScore":   &graphql.Field{Type: graphql.Float},
		"problemCount":    &graphql.Field{Type: graphql.Int},
		"createdAt":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"updatedAt":       &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
	},
})

var TrendSnapshotType = graphql.NewObject(graphql.ObjectConfig{
	Name: "TrendSnapshot",
	Fields: graphql.Fields{
		"id":           &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
		"clusterId":    &graphql.Field{Type: graphql.ID},
		"label":        &graphql.Field{Type: graphql.String},
		"problemCount": &graphql.Field{Type: graphql.Int},
		"growthRate":   &graphql.Field{Type: graphql.Float},
		"windowStart":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"windowEnd":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"snapshotAt":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
	},
})

var ChatSessionType = graphql.NewObject(graphql.ObjectConfig{
	Name: "ChatSession",
	Fields: graphql.Fields{
		"id":        &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
		"createdAt": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"updatedAt": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
	},
})

var ChatMessageType = graphql.NewObject(graphql.ObjectConfig{
	Name: "ChatMessage",
	Fields: graphql.Fields{
		"id":        &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
		"sessionId": &graphql.Field{Type: graphql.NewNonNull(graphql.ID)},
		"role":      &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"content":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"sources":   &graphql.Field{Type: graphql.String},
		"createdAt": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
	},
})
var CategoryCountType = graphql.NewObject(graphql.ObjectConfig{
	Name: "CategoryCount",
	Fields: graphql.Fields{
		"category": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"count":    &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
	},
})

var ProblemConnectionType = graphql.NewObject(graphql.ObjectConfig{
	Name: "ProblemConnection",
	Fields: graphql.Fields{
		"items":      &graphql.Field{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(RawProblemType)))},
		"totalCount": &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		"page":       &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		"pageSize":   &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
	},
})
var ClusterConnectionType = graphql.NewObject(graphql.ObjectConfig{
	Name: "ClusterConnection",
	Fields: graphql.Fields{
		"items":      &graphql.Field{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(ProblemClusterType)))},
		"totalCount": &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		"page":       &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
		"pageSize":   &graphql.Field{Type: graphql.NewNonNull(graphql.Int)},
	},
})

var ProblemFilterInput = graphql.NewInputObject(graphql.InputObjectConfig{
	Name: "ProblemFilter",
	Fields: graphql.InputObjectConfigFieldMap{
		"source":   &graphql.InputObjectFieldConfig{Type: graphql.String},
		"category": &graphql.InputObjectFieldConfig{Type: graphql.String},
		"tags":     &graphql.InputObjectFieldConfig{Type: graphql.NewList(graphql.NewNonNull(graphql.String))},
		"dateFrom": &graphql.InputObjectFieldConfig{Type: graphql.String},
		"dateTo":   &graphql.InputObjectFieldConfig{Type: graphql.String},
		"minScore": &graphql.InputObjectFieldConfig{Type: graphql.Int},
	},
})
