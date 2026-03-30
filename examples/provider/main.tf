terraform {
  required_providers {
    mijnhost = {
      source  = "tieum/mijnhost"
      version = "~> 0.1"
    }
  }
}

provider "mijnhost" {
  # api_key = "your-api-key"
  # Or set the MIJNHOST_API_KEY environment variable.
}
