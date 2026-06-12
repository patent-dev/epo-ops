.PHONY: generate test test-integration check-integration refresh-fixtures lint fmt coverage tidy examples

# generate re-applies the OpenAPI fixes (if a script is present) and regenerates
# the typed client via the //go:generate directives.
generate:
	@if [ -x scripts/fix-openapi.sh ]; then ./scripts/fix-openapi.sh; \
	elif [ -x scripts/convert-openapi.sh ]; then ./scripts/convert-openapi.sh; fi
	go generate ./...

test:
	go test -race -count=1 ./...

test-integration:
	go test -race -count=1 -tags=integration ./...

# check-integration verifies every exported Client method has a per-endpoint
# TestIntegration<Method> live test (see scripts/check-integration-coverage.sh).
check-integration:
	./scripts/check-integration-coverage.sh

lint:
	golangci-lint run

fmt:
	gofmt -w .

coverage:
	go test -race -count=1 -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1

tidy:
	go mod tidy

# examples runs the demo against the live API to refresh demo/examples (recorded
# request/response pairs). Requires the API credentials in the environment.
# The weekly response-watch workflow runs this and diffs the result.
examples:
	cd demo && go run .

# refresh-fixtures deliberately re-copies the live demo/examples recordings over
# the committed golden XML fixtures in testdata/. Run it (after `make examples`)
# ONLY when a human intends to update the goldens to a newer real response.
#
# The deterministic fixture tests (decode_fixtures_test.go) read the COMMITTED
# testdata/*.xml copies, never demo/examples, so re-running the demo does not
# change test behaviour until the goldens are refreshed here and the diff is
# reviewed and committed. The mapping below mirrors completenessCases.
refresh-fixtures:
	cp demo/examples/get_abstract/response.xml                  testdata/abstract.xml
	cp demo/examples/get_biblio/response.xml                    testdata/biblio.xml
	cp demo/examples/get_claims/response.xml                    testdata/claims.xml
	cp demo/examples/get_description/response.xml               testdata/description.xml
	cp demo/examples/get_family/response.xml                    testdata/family.xml
	cp demo/examples/get_family_with_biblio/response.xml        testdata/family-biblio.xml
	cp demo/examples/get_legal/response.xml                     testdata/legal.xml
	cp demo/examples/search/response.xml                        testdata/search.xml
	cp demo/examples/get_published_equivalents/response.xml     testdata/equivalents.xml
	cp demo/examples/get_register_events/response.xml           testdata/register-events.xml
	cp demo/examples/get_register_biblio/response.xml           testdata/register-biblio.xml
	cp demo/examples/get_register_procedural_steps/response.xml testdata/register-procedural-steps.xml
	cp demo/examples/get_register_unip/response.xml             testdata/register-unip.xml
	cp demo/examples/convert_patent_number_epodoc/response.xml  testdata/number-conversion.xml
	cp demo/examples/get_classification_schema/response.xml     testdata/classification-schema.xml
	@echo "testdata/*.xml updated from demo/examples; review the diff before committing."
	@echo "note: testdata/image-inquiry.xml is a curated XML fixture (the demo records"
	@echo "      the image-inquiry endpoint as a text summary, not raw XML); refresh it"
	@echo "      by hand from a GetImageInquiry raw response if needed."
