package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

/*
	we create a custom span here, the var tracer created is usually created per package
	don't think abt exporting it from internal/otel because then all packages would be dependent on it
*/

var tracer = otel.Tracer("shazam/cmd/api/tracer-working")

func SomeLogic(w http.ResponseWriter, r *http.Request) {

	/*
		give spans the context of the request (usual way)
		give them a name to identify, we call them "logic" here
		the first return value is context itself, which is useful when you want to create nested spans
	*/
	_, span := tracer.Start(r.Context(), "manual")

	statusCode := http.StatusOK
	defer func() {
		/*
		 we ASSUME here everything went well so directly write status = ok
		 set a predefined attribute (HTTPResponseStatusCode) from semconv library (try to use as far as possible)
		 set a custom attribute key-value pair here {"msg": "logic run"}
		*/
		span.SetAttributes(
			semconv.HTTPResponseStatusCode(statusCode),
			attribute.String("msg", "manual span logic run"),
		)
		span.End()
	}()

	time.Sleep(time.Second * 5)
	log.Printf("hello from shazam")

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(statusCode)
	fmt.Fprint(w, "Hello, you've reached the Go server!")

}

/*
	we can create child spans of a created span
	just use the span of the created span, during creation of the child span
*/
func ChildLogic(w http.ResponseWriter, r *http.Request) {
	ctx, parentSpan := tracer.Start(r.Context(), "childLogic.parent")
	defer parentSpan.End()

	statusCode := http.StatusOK
	msg, err := runChildWork(ctx)
	if err != nil {
		statusCode = http.StatusInternalServerError
		msg = "child logic failed"
	}

	parentSpan.SetAttributes(
		semconv.HTTPResponseStatusCode(statusCode),
		attribute.String("msg", "child logic run"),
	)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(statusCode)
	fmt.Fprint(w, msg)
}

// continued of the child span creation
func runChildWork(ctx context.Context) (string, error) {
	_, childSpan := tracer.Start(ctx, "childLogic.childWork")
	defer childSpan.End()

	time.Sleep(300 * time.Millisecond)
	childSpan.SetAttributes(attribute.String("work.type", "simulated"))
	return "Child span created under parent span", nil
}

func AutoSpan(w http.ResponseWriter, r *http.Request) {
	// No manual tracer.Start call here.
	// otelhttp middleware creates the server span automatically for this request.
	time.Sleep(120 * time.Millisecond)
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Auto span route: only otelhttp server span is created")
}
