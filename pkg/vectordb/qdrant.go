package vectordb

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/memtensor/memgos/pkg/types"
)

// QdrantVectorDB implements the BaseVectorDB interface for Qdrant
type QdrantVectorDB struct {
	*BaseVectorDBImpl
	client     qdrant.QdrantClient
	conn       *grpc.ClientConn
	config     *QdrantConfig
}

// NewQdrantVectorDB creates a new Qdrant vector database client
func NewQdrantVectorDB(config *QdrantConfig) (*QdrantVectorDB, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid qdrant config: %w", err)
	}

	baseConfig := &VectorDBConfig{
		Backend: types.BackendQdrant,
		Qdrant:  config,
	}

	return &QdrantVectorDB{
		BaseVectorDBImpl: NewBaseVectorDBImpl(baseConfig),
		config:           config,
	}, nil
}

// Connect establishes connection to Qdrant
func (q *QdrantVectorDB) Connect(ctx context.Context) error {
	q.SetConnectionStatus(ConnectionStatusConnecting)

	address := fmt.Sprintf("%s:%d", q.config.Host, q.config.Port)
	
	// Setup connection options
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(q.config.ConnectionTimeout),
		grpc.WithBlock(),
	}

	// Establish connection with retry
	operation := func() error {
		conn, err := grpc.DialContext(ctx, address, opts...)
		if err != nil {
			return fmt.Errorf("failed to connect to qdrant at %s: %w", address, err)
		}
		
		q.conn = conn
		q.client = qdrant.NewQdrantClient(conn)
		return nil
	}

	retryConfig := backoff.NewExponentialBackOff()
	retryConfig.MaxElapsedTime = time.Duration(q.config.MaxRetries) * q.config.RetryInterval
	
	if err := backoff.Retry(operation, retryConfig); err != nil {
		q.SetConnectionStatus(ConnectionStatusError)
		return fmt.Errorf("failed to connect to qdrant after retries: %w", err)
	}

	q.SetConnectionStatus(ConnectionStatusConnected)
	return nil
}

// Disconnect closes the connection to Qdrant
func (q *QdrantVectorDB) Disconnect(ctx context.Context) error {
	if q.conn != nil {
		err := q.conn.Close()
		q.conn = nil
		q.client = nil
		q.SetConnectionStatus(ConnectionStatusDisconnected)
		return err
	}
	return nil
}

// HealthCheck performs a health check on the Qdrant connection
func (q *QdrantVectorDB) HealthCheck(ctx context.Context) error {
	if !q.IsConnected() {
		return fmt.Errorf("not connected to qdrant")
	}

	// Perform health check by listing collections
	collectionsClient := qdrant.NewCollectionsClient(q.conn)
	_, err := collectionsClient.List(ctx, &qdrant.ListCollectionsRequest{})
	if err != nil {
		q.SetConnectionStatus(ConnectionStatusError)
		return fmt.Errorf("qdrant health check failed: %w", err)
	}

	q.SetLastHealthCheck(time.Now())
	return nil
}

// CreateCollection creates a new collection in Qdrant
func (q *QdrantVectorDB) CreateCollection(ctx context.Context, config *CollectionConfig) error {
	if err := q.ValidateCollection(config); err != nil {
		return err
	}

	// Check if collection already exists
	exists, err := q.CollectionExists(ctx, config.Name)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}
	if exists {
		return fmt.Errorf("collection %s already exists", config.Name)
	}

	// Convert distance metric
	distance, err := q.convertDistanceMetric(config.Distance)
	if err != nil {
		return fmt.Errorf("invalid distance metric: %w", err)
	}

	// Create vectors config
	vectorsConfig := &qdrant.VectorsConfig{
		Config: &qdrant.VectorsConfig_Params{
			Params: &qdrant.VectorParams{
				Size:     uint64(config.VectorSize),
				Distance: distance,
			},
		},
	}

	// Create collection request
	req := &qdrant.CreateCollection{
		CollectionName: config.Name,
		VectorsConfig:  vectorsConfig,
		OnDiskPayload:  &config.OnDiskPayload,
	}

	// Add quantization config if provided
	if config.Quantization != nil {
		if config.Quantization.Scalar != nil {
			req.QuantizationConfig = &qdrant.QuantizationConfig{
				Quantization: &qdrant.QuantizationConfig_Scalar{
					Scalar: &qdrant.ScalarQuantization{
						Type:      qdrant.QuantizationType_Int8,
						Quantile:  &config.Quantization.Scalar.Quantile,
						AlwaysRam: &config.Quantization.Scalar.AlwaysRam,
					},
				},
			}
		} else if config.Quantization.Binary != nil {
			req.QuantizationConfig = &qdrant.QuantizationConfig{
				Quantization: &qdrant.QuantizationConfig_Binary{
					Binary: &qdrant.BinaryQuantization{
						AlwaysRam: &config.Quantization.Binary.AlwaysRam,
					},
				},
			}
		}
	}

	// Execute create collection
	collectionsClient := qdrant.NewCollectionsClient(q.conn)
	_, err = collectionsClient.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to create collection %s: %w", config.Name, err)
	}

	return nil
}

// ListCollections lists all collections in Qdrant
func (q *QdrantVectorDB) ListCollections(ctx context.Context) ([]string, error) {
	collectionsClient := qdrant.NewCollectionsClient(q.conn)
	resp, err := collectionsClient.List(ctx, &qdrant.ListCollectionsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	collections := make([]string, len(resp.Collections))
	for i, collection := range resp.Collections {
		collections[i] = collection.Name
	}

	return collections, nil
}

// DeleteCollection deletes a collection from Qdrant
func (q *QdrantVectorDB) DeleteCollection(ctx context.Context, collectionName string) error {
	if collectionName == "" {
		return fmt.Errorf("collection name cannot be empty")
	}

	collectionsClient := qdrant.NewCollectionsClient(q.conn)
	_, err := collectionsClient.Delete(ctx, &qdrant.DeleteCollection{
		CollectionName: collectionName,
	})
	if err != nil {
		return fmt.Errorf("failed to delete collection %s: %w", collectionName, err)
	}

	return nil
}

// CollectionExists checks if a collection exists in Qdrant
func (q *QdrantVectorDB) CollectionExists(ctx context.Context, collectionName string) (bool, error) {
	collectionsClient := qdrant.NewCollectionsClient(q.conn)
	resp, err := collectionsClient.CollectionExists(ctx, &qdrant.CollectionExistsRequest{
		CollectionName: collectionName,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check collection existence: %w", err)
	}

	return resp.Result.Exists, nil
}

// GetCollectionInfo retrieves information about a collection
func (q *QdrantVectorDB) GetCollectionInfo(ctx context.Context, collectionName string) (*CollectionInfo, error) {
	collectionsClient := qdrant.NewCollectionsClient(q.conn)
	resp, err := collectionsClient.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: collectionName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get collection info: %w", err)
	}

	info := &CollectionInfo{
		Name:            collectionName,
		PointsCount:     int64(*resp.Result.PointsCount),
		IndexedVectors:  int64(*resp.Result.IndexedVectorsCount),
		Status:          resp.Result.Status.String(),
		OptimizerStatus: resp.Result.OptimizerStatus.String(),
	}

	// Extract vector configuration
	if resp.Result.Config != nil && resp.Result.Config.Params != nil {
		if vectorConfig := resp.Result.Config.Params.VectorsConfig; vectorConfig != nil {
			if params := vectorConfig.GetParams(); params != nil {
				info.VectorSize = int(params.Size)
				info.Distance = q.convertDistanceFromPb(params.Distance)
			}
		}
	}

	return info, nil
}

// Search performs vector similarity search
func (q *QdrantVectorDB) Search(ctx context.Context, collectionName string, vector []float32, limit int, filters map[string]interface{}) (VectorDBItemList, error) {
	if len(vector) == 0 {
		return nil, fmt.Errorf("search vector cannot be empty")
	}

	if limit <= 0 {
		limit = q.config.DefaultTopK
	}

	// Convert filters to Qdrant filter
	filter, err := q.convertFilters(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to convert filters: %w", err)
	}

	// Create search request
	req := &qdrant.SearchPoints{
		CollectionName: collectionName,
		Vector:         vector,
		Limit:          uint64(limit),
		Filter:         filter,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
		WithVectors:    &qdrant.WithVectorsSelector{SelectorOptions: &qdrant.WithVectorsSelector_Enable{Enable: false}},
	}

	// Add score threshold if configured
	if q.config.ScoreThreshold > 0 {
		req.ScoreThreshold = &q.config.ScoreThreshold
	}

	// Execute search
	pointsClient := qdrant.NewPointsClient(q.conn)
	resp, err := pointsClient.Search(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to search points: %w", err)
	}

	// Convert results to VectorDBItems
	items := make(VectorDBItemList, len(resp.Result))
	for i, point := range resp.Result {
		item, err := q.convertScoredPointToItem(point)
		if err != nil {
			return nil, fmt.Errorf("failed to convert point to item: %w", err)
		}
		items[i] = item
	}

	return items, nil
}

// Add inserts new vectors into the collection
func (q *QdrantVectorDB) Add(ctx context.Context, collectionName string, items VectorDBItemList) error {
	return q.upsertPoints(ctx, collectionName, items, false)
}

// Update updates existing vectors in the collection
func (q *QdrantVectorDB) Update(ctx context.Context, collectionName string, items VectorDBItemList) error {
	return q.upsertPoints(ctx, collectionName, items, false)
}

// Upsert inserts or updates vectors in the collection
func (q *QdrantVectorDB) Upsert(ctx context.Context, collectionName string, items VectorDBItemList) error {
	return q.upsertPoints(ctx, collectionName, items, true)
}

// Delete removes vectors from the collection
func (q *QdrantVectorDB) Delete(ctx context.Context, collectionName string, ids []string) error {
	if len(ids) == 0 {
		return fmt.Errorf("ids list cannot be empty")
	}

	// Convert string IDs to PointIds
	pointIds := make([]*qdrant.PointId, len(ids))
	for i, id := range ids {
		pointIds[i] = &qdrant.PointId{
			PointIdOptions: &qdrant.PointId_Uuid{Uuid: id},
		}
	}

	// Create delete request
	req := &qdrant.DeletePoints{
		CollectionName: collectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{Ids: pointIds},
			},
		},
	}

	// Execute delete
	pointsClient := qdrant.NewPointsClient(q.conn)
	_, err := pointsClient.Delete(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to delete points: %w", err)
	}

	return nil
}

// GetByID retrieves a single vector by ID
func (q *QdrantVectorDB) GetByID(ctx context.Context, collectionName string, id string) (*VectorDBItem, error) {
	items, err := q.GetByIDs(ctx, collectionName, []string{id})
	if err != nil {
		return nil, err
	}
	
	if len(items) == 0 {
		return nil, &types.MemGOSError{
			Type:    types.ErrorTypeNotFound,
			Message: fmt.Sprintf("vector with ID %s not found", id),
			Code:    "VECTOR_NOT_FOUND",
		}
	}
	
	return items[0], nil
}

// GetByIDs retrieves multiple vectors by their IDs
func (q *QdrantVectorDB) GetByIDs(ctx context.Context, collectionName string, ids []string) (VectorDBItemList, error) {
	if len(ids) == 0 {
		return VectorDBItemList{}, nil
	}

	// Convert string IDs to PointIds
	pointIds := make([]*qdrant.PointId, len(ids))
	for i, id := range ids {
		pointIds[i] = &qdrant.PointId{
			PointIdOptions: &qdrant.PointId_Uuid{Uuid: id},
		}
	}

	// Create get request
	req := &qdrant.GetPoints{
		CollectionName: collectionName,
		Ids:            pointIds,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
		WithVectors:    &qdrant.WithVectorsSelector{SelectorOptions: &qdrant.WithVectorsSelector_Enable{Enable: true}},
	}

	// Execute get
	pointsClient := qdrant.NewPointsClient(q.conn)
	resp, err := pointsClient.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get points: %w", err)
	}

	// Convert results to VectorDBItems
	items := make(VectorDBItemList, len(resp.Result))
	for i, point := range resp.Result {
		item, err := q.convertPointToItem(point)
		if err != nil {
			return nil, fmt.Errorf("failed to convert point to item: %w", err)
		}
		items[i] = item
	}

	return items, nil
}

// GetByFilter retrieves vectors matching a filter
func (q *QdrantVectorDB) GetByFilter(ctx context.Context, collectionName string, filters map[string]interface{}) (VectorDBItemList, error) {
	filter, err := q.convertFilters(filters)
	if err != nil {
		return nil, fmt.Errorf("failed to convert filters: %w", err)
	}

	req := &qdrant.ScrollPoints{
		CollectionName: collectionName,
		Filter:         filter,
		Limit:          &[]uint32{uint32(q.config.SearchLimit)}[0],
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
		WithVectors:    &qdrant.WithVectorsSelector{SelectorOptions: &qdrant.WithVectorsSelector_Enable{Enable: true}},
	}

	pointsClient := qdrant.NewPointsClient(q.conn)
	resp, err := pointsClient.Scroll(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to scroll points: %w", err)
	}

	items := make(VectorDBItemList, len(resp.Result))
	for i, point := range resp.Result {
		item, err := q.convertPointToItem(point)
		if err != nil {
			return nil, fmt.Errorf("failed to convert point to item: %w", err)
		}
		items[i] = item
	}

	return items, nil
}

// GetAll retrieves all vectors from the collection with pagination
func (q *QdrantVectorDB) GetAll(ctx context.Context, collectionName string, limit int, offset int) (VectorDBItemList, error) {
	if limit <= 0 {
		limit = q.config.SearchLimit
	}

	req := &qdrant.ScrollPoints{
		CollectionName: collectionName,
		Limit:          &[]uint32{uint32(limit)}[0],
		Offset:         &qdrant.PointId{PointIdOptions: &qdrant.PointId_Uuid{Uuid: fmt.Sprintf("%d", offset)}},
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
		WithVectors:    &qdrant.WithVectorsSelector{SelectorOptions: &qdrant.WithVectorsSelector_Enable{Enable: true}},
	}

	pointsClient := qdrant.NewPointsClient(q.conn)
	resp, err := pointsClient.Scroll(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to scroll points: %w", err)
	}

	items := make(VectorDBItemList, len(resp.Result))
	for i, point := range resp.Result {
		item, err := q.convertPointToItem(point)
		if err != nil {
			return nil, fmt.Errorf("failed to convert point to item: %w", err)
		}
		items[i] = item
	}

	return items, nil
}

// Count returns the number of vectors in the collection matching optional filters
func (q *QdrantVectorDB) Count(ctx context.Context, collectionName string, filters map[string]interface{}) (int64, error) {
	filter, err := q.convertFilters(filters)
	if err != nil {
		return 0, fmt.Errorf("failed to convert filters: %w", err)
	}

	req := &qdrant.CountPoints{
		CollectionName: collectionName,
		Filter:         filter,
		Exact:          &[]bool{true}[0],
	}

	pointsClient := qdrant.NewPointsClient(q.conn)
	resp, err := pointsClient.Count(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("failed to count points: %w", err)
	}

	return int64(resp.Result.Count), nil
}

// BatchAdd performs batch insertion of vectors
func (q *QdrantVectorDB) BatchAdd(ctx context.Context, collectionName string, items VectorDBItemList, batchSize int) error {
	return q.batchOperation(ctx, collectionName, items, nil, batchSize, OperationAdd)
}

// BatchUpdate performs batch update of vectors
func (q *QdrantVectorDB) BatchUpdate(ctx context.Context, collectionName string, items VectorDBItemList, batchSize int) error {
	return q.batchOperation(ctx, collectionName, items, nil, batchSize, OperationUpdate)
}

// BatchDelete performs batch deletion of vectors
func (q *QdrantVectorDB) BatchDelete(ctx context.Context, collectionName string, ids []string, batchSize int) error {
	return q.batchOperation(ctx, collectionName, nil, ids, batchSize, OperationDelete)
}

// GetStats returns statistics for the collection
func (q *QdrantVectorDB) GetStats(ctx context.Context, collectionName string) (*DatabaseStats, error) {
	info, err := q.GetCollectionInfo(ctx, collectionName)
	if err != nil {
		return nil, err
	}

	return &DatabaseStats{
		CollectionName: collectionName,
		PointsCount:    info.PointsCount,
		IndexedVectors: info.IndexedVectors,
		PayloadSchema:  make(map[string]string), // This would need more detailed implementation
	}, nil
}

// Helper methods

func (q *QdrantVectorDB) convertDistanceMetric(distance string) (qdrant.Distance, error) {
	switch distance {
	case "cosine":
		return qdrant.Distance_Cosine, nil
	case "euclidean":
		return qdrant.Distance_Euclid, nil
	case "dot":
		return qdrant.Distance_Dot, nil
	default:
		return 0, fmt.Errorf("unsupported distance metric: %s", distance)
	}
}

func (q *QdrantVectorDB) convertDistanceFromPb(distance qdrant.Distance) string {
	switch distance {
	case qdrant.Distance_Cosine:
		return "cosine"
	case qdrant.Distance_Euclid:
		return "euclidean"
	case qdrant.Distance_Dot:
		return "dot"
	default:
		return "unknown"
	}
}

func (q *QdrantVectorDB) convertFilters(filters map[string]interface{}) (*qdrant.Filter, error) {
	if len(filters) == 0 {
		return nil, nil
	}

	// Simple filter conversion - this can be extended for more complex filtering
	var conditions []*qdrant.Condition
	for key, value := range filters {
		condition := &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key: key,
					Match: &qdrant.Match{
						MatchValue: &qdrant.Match_Keyword{
							Keyword: fmt.Sprintf("%v", value),
						},
					},
				},
			},
		}
		conditions = append(conditions, condition)
	}

	return &qdrant.Filter{
		Must: conditions,
	}, nil
}

func (q *QdrantVectorDB) convertPointToItem(point *qdrant.RetrievedPoint) (*VectorDBItem, error) {
	id := point.Id.GetUuid()
	if id == "" {
		id = fmt.Sprintf("%d", point.Id.GetNum())
	}

	item := &VectorDBItem{
		ID: id,
	}

	// Extract vector if present
	if vectors := point.Vectors; vectors != nil {
		if vectorData := vectors.GetVector(); vectorData != nil {
			item.Vector = vectorData.Data
		}
	}

	// Extract payload if present
	if payload := point.Payload; payload != nil {
		item.Payload = make(map[string]interface{})
		for key, value := range payload {
			item.Payload[key] = q.convertPayloadValue(value)
		}
	}

	return item, nil
}

func (q *QdrantVectorDB) convertScoredPointToItem(point *qdrant.ScoredPoint) (*VectorDBItem, error) {
	id := point.Id.GetUuid()
	if id == "" {
		id = fmt.Sprintf("%d", point.Id.GetNum())
	}

	item := &VectorDBItem{
		ID:    id,
		Score: &point.Score,
	}

	// Extract vector if present
	if vectors := point.Vectors; vectors != nil {
		if vectorData := vectors.GetVector(); vectorData != nil {
			item.Vector = vectorData.Data
		}
	}

	// Extract payload if present
	if payload := point.Payload; payload != nil {
		item.Payload = make(map[string]interface{})
		for key, value := range payload {
			item.Payload[key] = q.convertPayloadValue(value)
		}
	}

	return item, nil
}

func (q *QdrantVectorDB) convertPayloadValue(value *qdrant.Value) interface{} {
	switch v := value.Kind.(type) {
	case *qdrant.Value_StringValue:
		return v.StringValue
	case *qdrant.Value_IntegerValue:
		return v.IntegerValue
	case *qdrant.Value_DoubleValue:
		return v.DoubleValue
	case *qdrant.Value_BoolValue:
		return v.BoolValue
	case *qdrant.Value_ListValue:
		result := make([]interface{}, len(v.ListValue.Values))
		for i, val := range v.ListValue.Values {
			result[i] = q.convertPayloadValue(val)
		}
		return result
	case *qdrant.Value_StructValue:
		result := make(map[string]interface{})
		for key, val := range v.StructValue.Fields {
			result[key] = q.convertPayloadValue(val)
		}
		return result
	default:
		return nil
	}
}

func (q *QdrantVectorDB) convertToPayloadValue(value interface{}) *qdrant.Value {
	switch v := value.(type) {
	case string:
		return &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: v}}
	case int:
		return &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(v)}}
	case int64:
		return &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: v}}
	case float32:
		return &qdrant.Value{Kind: &qdrant.Value_DoubleValue{DoubleValue: float64(v)}}
	case float64:
		return &qdrant.Value{Kind: &qdrant.Value_DoubleValue{DoubleValue: v}}
	case bool:
		return &qdrant.Value{Kind: &qdrant.Value_BoolValue{BoolValue: v}}
	case []interface{}:
		values := make([]*qdrant.Value, len(v))
		for i, val := range v {
			values[i] = q.convertToPayloadValue(val)
		}
		return &qdrant.Value{Kind: &qdrant.Value_ListValue{ListValue: &qdrant.ListValue{Values: values}}}
	case map[string]interface{}:
		fields := make(map[string]*qdrant.Value)
		for key, val := range v {
			fields[key] = q.convertToPayloadValue(val)
		}
		return &qdrant.Value{Kind: &qdrant.Value_StructValue{StructValue: &qdrant.Struct{Fields: fields}}}
	default:
		return &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: fmt.Sprintf("%v", v)}}
	}
}

func (q *QdrantVectorDB) upsertPoints(ctx context.Context, collectionName string, items VectorDBItemList, upsert bool) error {
	if err := q.ValidateItems(items); err != nil {
		return err
	}

	// Convert items to Qdrant points
	points := make([]*qdrant.PointStruct, len(items))
	for i, item := range items {
		// Generate UUID if not provided
		id := item.ID
		if id == "" {
			id = uuid.New().String()
		}

		point := &qdrant.PointStruct{
			Id: &qdrant.PointId{
				PointIdOptions: &qdrant.PointId_Uuid{Uuid: id},
			},
			Vectors: &qdrant.Vectors{
				VectorsOptions: &qdrant.Vectors_Vector{
					Vector: &qdrant.Vector{Data: item.Vector},
				},
			},
		}

		// Add payload if present
		if item.Payload != nil {
			payload := make(map[string]*qdrant.Value)
			for key, value := range item.Payload {
				payload[key] = q.convertToPayloadValue(value)
			}
			point.Payload = payload
		}

		points[i] = point
	}

	// Create upsert request
	req := &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points:         points,
	}

	// Execute upsert
	pointsClient := qdrant.NewPointsClient(q.conn)
	_, err := pointsClient.Upsert(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to upsert points: %w", err)
	}

	return nil
}

func (q *QdrantVectorDB) batchOperation(ctx context.Context, collectionName string, items VectorDBItemList, ids []string, batchSize int, operation VectorOperation) error {
	if batchSize <= 0 {
		batchSize = q.config.BatchSize
	}

	switch operation {
	case OperationAdd, OperationUpdate, OperationUpsert:
		for i := 0; i < len(items); i += batchSize {
			end := i + batchSize
			if end > len(items) {
				end = len(items)
			}
			
			batch := items[i:end]
			if err := q.upsertPoints(ctx, collectionName, batch, operation == OperationUpsert); err != nil {
				return fmt.Errorf("batch operation failed at batch %d: %w", i/batchSize, err)
			}
		}
	case OperationDelete:
		for i := 0; i < len(ids); i += batchSize {
			end := i + batchSize
			if end > len(ids) {
				end = len(ids)
			}
			
			batch := ids[i:end]
			if err := q.Delete(ctx, collectionName, batch); err != nil {
				return fmt.Errorf("batch delete failed at batch %d: %w", i/batchSize, err)
			}
		}
	}

	return nil
}