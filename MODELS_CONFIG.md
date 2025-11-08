# Models Configuration

The application supports loading model configurations from a JSON file, allowing you to add new models without recompiling.

## Configuration File Location

The app looks for `models.json` in the following locations (in order):
1. Current directory (`./models.json`)
2. Explicit current directory (`./models.json`)
3. Parent directory (`../models.json`)
4. User config (`~/.config/ivrit-ai/models.json`)

## Configuration Format

```json
{
  "models": {
    "model-id": {
      "id": "huggingface-repo-id",
      "file": "remote-filename.bin",
      "localFileName": "local-filename.bin",
      "description": "Model description"
    }
  }
}
```

### Fields:

- **id** (required): The HuggingFace repository ID (e.g., `ivrit-ai/whisper-large-v3-ggml`)
- **file** (required): The filename in the HuggingFace repository
- **localFileName** (optional): The filename to use when saving locally. If not specified, uses `file`. Useful when multiple models have the same remote filename.
- **description** (optional): Human-readable description of the model

## Example Configuration

The default `models.json` includes:

```json
{
  "models": {
    "large-v3": {
      "id": "ivrit-ai/whisper-large-v3-ggml",
      "file": "ggml-model.bin",
      "localFileName": "ggml-large-v3-ivrit.bin",
      "description": "Ivrit.ai Large v3 - Best quality for Hebrew"
    },
    "turbo": {
      "id": "ivrit-ai/whisper-large-v3-turbo-ggml",
      "file": "ggml-model.bin",
      "localFileName": "ggml-large-v3-turbo-ivrit.bin",
      "description": "Ivrit.ai Turbo - Faster with good quality"
    },
    "base": {
      "id": "ggerganov/whisper.cpp",
      "file": "ggml-base.bin",
      "localFileName": "ggml-base.bin",
      "description": "Base model - Fast but lower quality"
    }
  }
}
```

## Adding a New Model

To add a new model, edit `models.json` and add a new entry:

```json
{
  "models": {
    "my-custom-model": {
      "id": "username/model-repo",
      "file": "ggml-model.bin",
      "localFileName": "my-custom-model.bin",
      "description": "My custom trained model"
    }
  }
}
```

Then rebuild the app or restart it. The new model will appear in the model dropdown.

## Fallback Behavior

If no `models.json` file is found, the application uses the hardcoded default models shown in the example above.
