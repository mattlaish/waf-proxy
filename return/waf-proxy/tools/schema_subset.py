"""Dependency-free checks for the exact JSON Schema subset used in this pack.
Not a full JSON Schema implementation, runtime test, or security qualification.
Run: python tools/validate_pack.py
"""
import copy
import datetime
import json
import math
import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
COUNT = {"positive_pages": 0, "negative_pages": 0, "control_responses": 0, "bindings": 0, "links": 0}

def read(path):
    return json.loads(path.read_text(encoding="utf-8-sig"))

def is_type(value, kind):
    return {
        "null": lambda: value is None,
        "boolean": lambda: isinstance(value, bool),
        "integer": lambda: isinstance(value, int) and not isinstance(value, bool),
        "number": lambda: isinstance(value, (int, float)) and not isinstance(value, bool) and math.isfinite(value),
        "string": lambda: isinstance(value, str),
        "array": lambda: isinstance(value, list),
        "object": lambda: isinstance(value, dict),
    }[kind]()

def valid(value, spec, root):
    if "$ref" in spec:
        target = root
        assert spec["$ref"].startswith("#/")
        for part in spec["$ref"][2:].split("/"):
            target = target[part]
        if not valid(value, target, root):
            return False
    if "type" in spec:
        kinds = spec["type"] if isinstance(spec["type"], list) else [spec["type"]]
        if not any(is_type(value, k) for k in kinds):
            return False
    if "const" in spec and value != spec["const"]:
        return False
    if "enum" in spec and value not in spec["enum"]:
        return False
    if "anyOf" in spec and not any(valid(value, v, root) for v in spec["anyOf"]):
        return False
    if "allOf" in spec and not all(valid(value, v, root) for v in spec["allOf"]):
        return False
    if "if" in spec:
        branch = spec.get("then", {}) if valid(value, spec["if"], root) else spec.get("else", {})
        if not valid(value, branch, root):
            return False
    if isinstance(value, dict):
        if any(k not in value for k in spec.get("required", [])):
            return False
        props = spec.get("properties", {})
        if spec.get("additionalProperties") is False and any(k not in props for k in value):
            return False
        if any(k in value and not valid(value[k], v, root) for k, v in props.items()):
            return False
    if isinstance(value, list):
        if len(value) < spec.get("minItems", 0) or len(value) > spec.get("maxItems", math.inf):
            return False
        if spec.get("uniqueItems") and len({json.dumps(v, sort_keys=True) for v in value}) != len(value):
            return False
        if "items" in spec and not all(valid(v, spec["items"], root) for v in value):
            return False
    if isinstance(value, str):
        if len(value) < spec.get("minLength", 0) or len(value) > spec.get("maxLength", math.inf):
            return False
        if "pattern" in spec and not re.search(spec["pattern"], value):
            return False
        if spec.get("format") == "date-time":
            try:
                dt = datetime.datetime.fromisoformat(value.replace("Z", "+00:00"))
                if dt.tzinfo is None:
                    return False
            except ValueError:
                return False
    if isinstance(value, (int, float)) and not isinstance(value, bool):
        if value < spec.get("minimum", -math.inf) or value > spec.get("maximum", math.inf):
            return False
    return True

KEYWORDS = {
    "$schema", "$id", "$ref", "$defs", "title", "type", "properties", "required",
    "additionalProperties", "items", "minItems", "maxItems", "uniqueItems",
    "minLength", "maxLength", "minimum", "maximum", "pattern", "format",
    "const", "enum", "anyOf", "allOf", "if", "then", "else"
}
def inspect_schema(spec, root):
    assert isinstance(spec, dict)
    assert not set(spec) - KEYWORDS, set(spec) - KEYWORDS
    if "$ref" in spec:
        target = root
        for part in spec["$ref"][2:].split("/"):
            target = target[part]
    if "pattern" in spec:
        re.compile(spec["pattern"])
    if "required" in spec:
        assert set(spec["required"]) <= set(spec.get("properties", {}))
    for key in ("properties", "$defs"):
        for child in spec.get(key, {}).values():
            inspect_schema(child, root)
    for key in ("items", "if", "then", "else"):
        if key in spec:
            inspect_schema(spec[key], root)
    for key in ("anyOf", "allOf"):
        for child in spec.get(key, []):
            inspect_schema(child, root)

