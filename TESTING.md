# Testing Guide

This document describes how to run tests for the ivrit.ai Hebrew Transcription application.

## Test Suite Overview

The test suite includes:

1. **Unit Tests** - Testing core functions and utilities
2. **Integration Tests** - Testing CLI functionality end-to-end
3. **Benchmarks** - Performance testing for critical functions

## Running Tests

**Important**: Tests require CGO and whisper.cpp headers. Use the test script:


### Run All Tests

```bash
./scripts/test.sh
```

### Run Specific Test Files

```bash
# Transcription logic tests
./scripts/test.sh -run TestContainsHebrew
./scripts/test.sh -run TestFormatOutput

# CLI tests
./scripts/test.sh -run TestCLI

# Model configuration tests
./scripts/test.sh -run TestLoadModelsConfig
```

### Run Tests with Coverage

```bash
./scripts/test.sh -cover
```

### Generate Coverage Report

```bash
# Generate coverage profile
./scripts/test.sh -coverprofile=coverage.out

# View coverage in browser
go tool cover -html=coverage.out
```

### Run Benchmarks

```bash
go test -bench=. -benchmem
```

## Test Files

### transcription_test.go

Tests for core transcription functionality:

- `TestContainsHebrew` - Hebrew text detection
- `TestIsVideoFile` - Video file identification
- `TestFormatTimestamp` - SRT/VTT timestamp formatting
- `TestFormatOutputText` - Text format output
- `TestFormatOutputJSON` - JSON format output
- `TestFormatOutputSRT` - SRT subtitle format
- `TestFormatOutputVTT` - VTT subtitle format
- `TestGetOptimalCPUThreads` - CPU thread optimization
- `BenchmarkContainsHebrew` - Performance test for Hebrew detection
- `BenchmarkFormatTimestamp` - Performance test for timestamp formatting

### cli_test.go

Functional tests for CLI mode:

**Prerequisites:** Build the binary first with `./build.sh`

- `TestCLIHelp` - Verify help output
- `TestCLINoInput` - Error handling for missing input
- `TestCLIInvalidFile` - Error handling for non-existent files
- `TestCLIInvalidModel` - Model name validation
- `TestCLIInvalidFormat` - Output format validation
- `TestAutoOutputFilename` - Automatic filename generation
- `TestModelValidation` - Model name validation logic
- `TestFormatValidation` - Format validation logic

### models_test.go

Tests for model configuration:

- `TestLoadModelsConfig` - Loading default configuration
- `TestLoadCustomModelsConfig` - Loading custom models.json
- `TestModelInfoValidation` - ModelInfo struct validation
- `TestDefaultModelConfiguration` - Default model setup
- `TestJSONUnmarshal` - JSON deserialization
- `TestJSONMarshal` - JSON serialization

## Test Data

Tests use minimal test data and temporary files:

- Temporary files are automatically cleaned up after tests
- No large model files or audio files are required for testing
- CLI tests skip automatically if binary is not built

## Continuous Integration

Tests run automatically via GitHub Actions on:
- Every push to main branch
- Every pull request
- Release builds

See `.github/workflows/ci.yml` for CI configuration.

## Writing New Tests

### Unit Test Template

```go
func TestMyFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected bool
    }{
        {"Test case 1", "input1", true},
        {"Test case 2", "input2", false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := MyFunction(tt.input)
            if result != tt.expected {
                t.Errorf("MyFunction(%q) = %v, expected %v",
                    tt.input, result, tt.expected)
            }
        })
    }
}
```

### Benchmark Template

```go
func BenchmarkMyFunction(b *testing.B) {
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        MyFunction("test input")
    }
}
```

## Troubleshooting

### CLI Tests Failing

**Symptom:** CLI tests are skipped or failing

**Solution:** Build the binary first:
```bash
./scripts/build.sh
```

### Coverage Too Low

**Symptom:** Coverage report shows low percentage

**Solution:** Add tests for uncovered code paths, especially:
- Error handling paths
- Edge cases
- Platform-specific code

### Test Timeout

**Symptom:** Tests timeout (especially on CI)

**Solution:** Use `t.Parallel()` for independent tests:
```go
func TestSomething(t *testing.T) {
    t.Parallel()
    // ... test code
}
```

### Flaky Tests

**Symptom:** Tests pass/fail inconsistently

**Common causes:**
- Race conditions
- Timing dependencies
- Unclean temporary files

**Solution:** Use `t.Cleanup()` for reliable cleanup:
```go
func TestWithFile(t *testing.T) {
    tmpFile, _ := os.CreateTemp("", "test-*")
    t.Cleanup(func() {
        os.Remove(tmpFile.Name())
    })
    // ... test code
}
```

## Test Guidelines

1. **Use table-driven tests** for multiple test cases
2. **Use descriptive test names** that explain what's being tested
3. **Clean up temporary files** with `defer` or `t.Cleanup()`
4. **Skip tests gracefully** when prerequisites aren't met
5. **Use `t.Parallel()`** for independent tests to speed up execution
6. **Test error paths** not just happy paths
7. **Keep tests focused** - one concept per test
8. **Avoid external dependencies** when possible (network, files, etc.)

## Performance Benchmarks

Expected performance targets:

| Function | Target | Notes |
|----------|--------|-------|
| `containsHebrew()` | < 100 ns/op | Simple string scanning |
| `FormatTimestamp()` | < 500 ns/op | String formatting |
| `FormatOutput()` | < 1 ms/op | Depends on segment count |

Run benchmarks to verify performance:
```bash
go test -bench=. -benchtime=10s
```

## Test Coverage Goals

Target coverage by component:

- Core utilities (transcription.go): **> 80%**
- CLI validation (cli.go): **> 70%**
- Model configuration (model_download.go): **> 60%**
- Overall project: **> 70%**

## Future Test Improvements

Potential areas for additional testing:

1. **Integration tests** with actual audio files (small samples)
2. **GUI tests** using Gio test framework
3. **Translation tests** with mocked Ollama responses
4. **Performance regression tests** for transcription speed
5. **Cross-platform tests** on different OS/architectures

---

For questions or suggestions about testing, please open an issue on GitHub.
