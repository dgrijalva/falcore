package falcore

import (
	"bytes"
	"container/list"
	"github.com/ngmoco/falcore/filter"
	"net/http"
	"testing"
	"time"
)

func validGetRequest() *filter.Request {
	tmp, _ := http.NewRequest("GET", "/hello", bytes.NewBuffer(make([]byte, 0)))
	return filter.NewRequest(tmp, nil, time.Now())
}

var stageTrack *list.List

func doStageTrack() {
	i := 0
	if stageTrack.Len() > 0 {
		i = stageTrack.Back().Value.(int)
	}
	stageTrack.PushBack(i + 1)
}

func sumFilter(req *filter.Request) *http.Response {
	doStageTrack()
	return nil
}

func sumResponseFilter(*filter.Request, *http.Response) {
	doStageTrack()
}

func successFilter(req *filter.Request) *http.Response {
	doStageTrack()
	return filter.SimpleResponse(req.HttpRequest, 200, nil, "OK")
}

func TestPipelineNoResponse(t *testing.T) {
	p := NewPipeline()

	stageTrack = list.New()
	f := filter.NewRequestFilter(sumFilter)

	p.Upstream.PushBack(f)
	p.Upstream.PushBack(f)
	p.Upstream.PushBack(f)

	//response := new(http.Response)
	response := p.execute(validGetRequest())

	if stageTrack.Len() != 3 {
		t.Fatalf("Wrong number of stages executed: %v expected %v", stageTrack.Len(), 3)
	}
	if sum, ok := stageTrack.Back().Value.(int); ok {
		if sum != 3 {
			t.Errorf("Pipeline stages did not complete %v expected %v", sum, 3)
		}
	}
	if response.StatusCode != 404 {
		t.Errorf("Pipeline response code wrong: %v expected %v", response.StatusCode, 404)
	}
}

func TestPipelineOKResponse(t *testing.T) {
	p := NewPipeline()

	stageTrack = list.New()
	f := filter.NewRequestFilter(sumFilter)

	p.Upstream.PushBack(f)
	p.Upstream.PushBack(f)
	p.Upstream.PushBack(filter.NewRequestFilter(successFilter))
	p.Upstream.PushBack(f)

	response := p.execute(validGetRequest())

	if stageTrack.Len() != 3 {
		t.Fatalf("Wrong number of stages executed: %v expected %v", stageTrack.Len(), 3)
	}
	if sum, ok := stageTrack.Back().Value.(int); ok {
		if sum != 3 {
			t.Errorf("Pipeline stages did not complete %v expected %v", sum, 3)
		}
	}
	if response.StatusCode != 200 {
		t.Errorf("Pipeline response code wrong: %v expected %v", response.StatusCode, 200)
	}
}

func TestPipelineResponseFilter(t *testing.T) {
	p := NewPipeline()

	stageTrack = list.New()
	f := filter.NewRequestFilter(sumFilter)

	p.Upstream.PushBack(f)
	p.Upstream.PushBack(filter.NewRequestFilter(successFilter))
	p.Upstream.PushBack(f)
	p.Downstream.PushBack(filter.NewResponseFilter(sumResponseFilter))
	p.Downstream.PushBack(filter.NewResponseFilter(sumResponseFilter))

	//response := new(http.Response)
	req := validGetRequest()
	response := p.execute(req)

	stages := 4
	// check basic execution
	if stageTrack.Len() != stages {
		t.Fatalf("Wrong number of stages executed: %v expected %v", stageTrack.Len(), stages)
	}
	if sum, ok := stageTrack.Back().Value.(int); ok {
		if sum != stages {
			t.Errorf("Pipeline stages did not complete %v expected %v", sum, stages)
		}
	}
	// check status
	if response.StatusCode != 200 {
		t.Errorf("Pipeline response code wrong: %v expected %v", response.StatusCode, 200)
	}
	req.FinishRequest()
	if req.Signature() != "DCA810F4" {
		t.Errorf("Signature failed: %v expected %v", req.Signature(), "DCA810F4")
	}
	if req.PipelineStageStats.Len() != stages {
		t.Errorf("PipelineStageStats incomplete: %v expected %v", req.PipelineStageStats.Len(), stages)
	}
	//req.Trace()

}

func TestPipelineStatsChecksum(t *testing.T) {
	p := NewPipeline()

	stageTrack = list.New()
	f := filter.NewRequestFilter(sumFilter)

	p.Upstream.PushBack(f)
	p.Upstream.PushBack(filter.NewRequestFilter(func(req *filter.Request) *http.Response {
		doStageTrack()
		req.CurrentStage.Status = 1
		return nil
	}))
	p.Upstream.PushBack(filter.NewRequestFilter(successFilter))
	p.Downstream.PushBack(filter.NewResponseFilter(sumResponseFilter))
	p.Downstream.PushBack(filter.NewResponseFilter(sumResponseFilter))

	//response := new(http.Response)
	req := validGetRequest()
	response := p.execute(req)

	stages := 5
	// check basic execution
	if stageTrack.Len() != stages {
		t.Fatalf("Wrong number of stages executed: %v expected %v", stageTrack.Len(), stages)
	}
	if sum, ok := stageTrack.Back().Value.(int); ok {
		if sum != stages {
			t.Errorf("Pipeline stages did not complete %v expected %v", sum, stages)
		}
	}
	// check status
	if response.StatusCode != 200 {
		t.Errorf("Pipeline response code wrong: %v expected %v", response.StatusCode, 200)
	}
	req.FinishRequest()
	if req.Signature() != "7C42487" {
		t.Errorf("Signature failed: %v expected %v", req.Signature(), "7C42487")
	}
	if req.PipelineStageStats.Len() != stages {
		t.Errorf("PipelineStageStats incomplete: %v expected %v", req.PipelineStageStats.Len(), stages)
	}
	//req.Trace()

}
