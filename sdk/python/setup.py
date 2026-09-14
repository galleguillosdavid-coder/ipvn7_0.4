from setuptools import setup, find_packages

setup(
    name="ipvn7",
    version="0.4.0",
    description="Official Python SDK for the ipvn7 Autonomous Sovereign Network OS (Post-Quantum & AI Agent Mesh)",
    long_description=open("README.md", encoding="utf-8").read(),
    long_description_content_type="text/markdown",
    author="galleguillosdavid-coder",
    url="https://github.com/galleguillosdavid-coder/ipvn7_0.4",
    packages=find_packages(),
    python_requires=">=3.8",
    classifiers=[
        "Development Status :: 4 - Beta",
        "Intended Audience :: Developers",
        "Topic :: System :: Networking",
        "Topic :: Security :: Cryptography",
        "Programming Language :: Python :: 3",
        "License :: OSI Approved :: Apache Software License",
    ],
)
