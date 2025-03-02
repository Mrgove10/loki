// Package marshal converts internal objects to loghttp model objects.  This
// package is designed to work with models in pkg/loghttp.
package marshal

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/websocket"
	jsoniter "github.com/json-iterator/go"

	"github.com/grafana/loki/pkg/loghttp"
	legacy "github.com/grafana/loki/pkg/loghttp/legacy"
	"github.com/grafana/loki/pkg/logproto"
	"github.com/grafana/loki/pkg/logqlmodel"
	"github.com/grafana/loki/pkg/storage/stores/index/stats"
	marshal_legacy "github.com/grafana/loki/pkg/util/marshal/legacy"
)

func WriteResponseJSON(r *http.Request, v any, w http.ResponseWriter) error {
	switch result := v.(type) {
	case logqlmodel.Result:
		version := loghttp.GetVersion(r.RequestURI)
		if version == loghttp.VersionV1 {
			return WriteQueryResponseJSON(result, w)
		}

		return marshal_legacy.WriteQueryResponseJSON(result, w)
	case *logproto.LabelResponse:
		version := loghttp.GetVersion(r.RequestURI)
		if version == loghttp.VersionV1 {
			return WriteLabelResponseJSON(*result, w)
		}

		return marshal_legacy.WriteLabelResponseJSON(*result, w)
	case *logproto.SeriesResponse:
		return WriteSeriesResponseJSON(*result, w)
	case *stats.Stats:
		return WriteIndexStatsResponseJSON(result, w)
	case *logproto.VolumeResponse:
		return WriteSeriesVolumeResponseJSON(result, w)
	}
	return fmt.Errorf("unknown response type %T", v)
}

// WriteQueryResponseJSON marshals the promql.Value to v1 loghttp JSON and then
// writes it to the provided io.Writer.
func WriteQueryResponseJSON(v logqlmodel.Result, w io.Writer) error {
	s := jsoniter.ConfigFastest.BorrowStream(w)
	defer jsoniter.ConfigFastest.ReturnStream(s)
	err := EncodeResult(v, s)
	if err != nil {
		return fmt.Errorf("could not write JSON response: %w", err)
	}
	s.WriteRaw("\n")
	return s.Flush()
}

// WriteLabelResponseJSON marshals a logproto.LabelResponse to v1 loghttp JSON
// and then writes it to the provided io.Writer.
func WriteLabelResponseJSON(l logproto.LabelResponse, w io.Writer) error {
	v1Response := loghttp.LabelResponse{
		Status: "success",
		Data:   l.GetValues(),
	}

	s := jsoniter.ConfigFastest.BorrowStream(w)
	defer jsoniter.ConfigFastest.ReturnStream(s)
	s.WriteVal(v1Response)
	s.WriteRaw("\n")
	return s.Flush()
}

// WebsocketWriter knows how to write message to a websocket connection.
type WebsocketWriter interface {
	WriteMessage(int, []byte) error
}

// WriteTailResponseJSON marshals the legacy.TailResponse to v1 loghttp JSON and
// then writes it to the provided connection.
func WriteTailResponseJSON(r legacy.TailResponse, c WebsocketWriter) error {
	v1Response, err := NewTailResponse(r)
	if err != nil {
		return err
	}
	data, err := jsoniter.Marshal(v1Response)
	if err != nil {
		return err
	}
	return c.WriteMessage(websocket.TextMessage, data)
}

// WriteSeriesResponseJSON marshals a logproto.SeriesResponse to v1 loghttp JSON and then
// writes it to the provided io.Writer.
func WriteSeriesResponseJSON(r logproto.SeriesResponse, w io.Writer) error {
	adapter := &seriesResponseAdapter{
		Status: "success",
		Data:   make([]map[string]string, 0, len(r.GetSeries())),
	}

	for _, series := range r.GetSeries() {
		adapter.Data = append(adapter.Data, series.GetLabels())
	}

	s := jsoniter.ConfigFastest.BorrowStream(w)
	defer jsoniter.ConfigFastest.ReturnStream(s)
	s.WriteVal(adapter)
	s.WriteRaw("\n")
	return s.Flush()
}

// This struct exists primarily because we can't specify a repeated map in proto v3.
// Otherwise, we'd use that + gogoproto.jsontag to avoid this layer of indirection
type seriesResponseAdapter struct {
	Status string              `json:"status"`
	Data   []map[string]string `json:"data"`
}

// WriteIndexStatsResponseJSON marshals a gatewaypb.Stats to JSON and then
// writes it to the provided io.Writer.
func WriteIndexStatsResponseJSON(r *stats.Stats, w io.Writer) error {
	s := jsoniter.ConfigFastest.BorrowStream(w)
	defer jsoniter.ConfigFastest.ReturnStream(s)
	s.WriteVal(r)
	s.WriteRaw("\n")
	return s.Flush()
}

// WriteSeriesVolumeResponseJSON marshals a logproto.VolumeResponse to JSON and then
// writes it to the provided io.Writer.
func WriteSeriesVolumeResponseJSON(r *logproto.VolumeResponse, w io.Writer) error {
	s := jsoniter.ConfigFastest.BorrowStream(w)
	defer jsoniter.ConfigFastest.ReturnStream(s)
	s.WriteVal(r)
	s.WriteRaw("\n")
	return s.Flush()
}
