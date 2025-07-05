"""
Setup for Temporal A2A SDK Python
"""
from setuptools import setup, find_packages

try:
    with open("README.md", "r", encoding="utf-8") as fh:
        long_description = fh.read()
except FileNotFoundError:
    long_description = "Python SDK for A2A protocol with Temporal - Zero Temporal knowledge required"

setup(
    name="temporal-a2a-sdk",
    version="0.1.0",
    author="Temporal A2A Gateway Team",
    description="Python SDK for A2A protocol with Temporal - Zero Temporal knowledge required",
    long_description=long_description,
    long_description_content_type="text/markdown",
    url="https://github.com/temporal-a2a-gateway/python-sdk",
    packages=find_packages(),
    classifiers=[
        "Development Status :: 3 - Alpha",
        "Intended Audience :: Developers",
        "Topic :: Software Development :: Libraries :: Python Modules",
        "License :: OSI Approved :: MIT License",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Programming Language :: Python :: 3.12",
    ],
    python_requires=">=3.8",
    install_requires=[
        "temporalio>=1.3.0",
        "aiohttp>=3.8.0",
    ],
    extras_require={
        "dev": [
            "pytest>=7.0",
            "pytest-asyncio>=0.20",
            "black>=22.0",
            "isort>=5.0",
            "mypy>=1.0",
        ]
    }
)