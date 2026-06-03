"""Minimal MCP-style example for MathModelAI.

This file is intentionally small. It documents the shape of a future MCP
server without pulling in a full service implementation.
"""

from dataclasses import dataclass


@dataclass
class LinearModel:
    coefficients: list[float]
    intercept: float = 0.0

    def predict(self, features: list[float]) -> float:
        if len(features) != len(self.coefficients):
            raise ValueError("features and coefficients must have the same length")
        return sum(c * x for c, x in zip(self.coefficients, features)) + self.intercept


if __name__ == "__main__":
    model = LinearModel(coefficients=[2.0, 3.0], intercept=1.0)
    print(model.predict([4.0, 5.0]))
