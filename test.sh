#!/bin/bash
set -e

echo "Running tests..."
go test -v -coverprofile=coverage.out

echo -e "\nTest Coverage Summary:"
go tool cover -func=coverage.out

echo -e "\nTo view detailed coverage report in browser, run:"
echo "go tool cover -html=coverage.out" 