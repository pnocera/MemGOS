# MemGOS MCP Server Examples

This document provides practical examples of using the MemGOS MCP server for various use cases and workflows.

## 🚀 Basic Examples

### Example 1: Setting Up a Research Knowledge Base

This example shows how to create and populate a research-focused memory system.

```bash
# Start the MCP server
export MEMGOS_API_TOKEN="your-api-token"
./memgos-mcp --api-url http://localhost:8080 --user researcher --session research-session
```

#### Step 1: Create Research Cubes

```json
// Register AI research cube
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "register_cube",
    "arguments": {
      "cube_id": "ai-research",
      "name": "AI Research Papers",
      "description": "Collection of AI and ML research findings"
    }
  }
}

// Register project cube
{
  "jsonrpc": "2.0", 
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "register_cube",
    "arguments": {
      "cube_id": "project-alpha",
      "name": "Project Alpha Notes",
      "description": "Notes and findings for Project Alpha"
    }
  }
}
```

#### Step 2: Add Research Content

```json
// Add paper summary
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "add_memory",
    "arguments": {
      "memory_content": "Attention Is All You Need (Vaswani et al., 2017): Introduced the Transformer architecture, replacing recurrent layers with self-attention mechanisms. Key innovations: multi-head attention, positional encoding, and parallelizable training. Achieved state-of-the-art results on machine translation tasks.",
      "mem_cube_id": "ai-research",
      "metadata": {
        "paper_title": "Attention Is All You Need",
        "authors": "Vaswani et al.",
        "year": 2017,
        "category": "architecture",
        "importance": "foundational"
      }
    }
  }
}

// Add implementation note
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "add_memory",
    "arguments": {
      "memory_content": "Implemented multi-head attention in PyTorch for Project Alpha. Key insight: scaling by sqrt(d_k) is crucial for gradient stability. Performance improved 25% over LSTM baseline on our dataset.",
      "mem_cube_id": "project-alpha",
      "metadata": {
        "implementation": "pytorch",
        "performance_gain": "25%",
        "date": "2024-01-15"
      }
    }
  }
}
```

#### Step 3: Query and Chat

```json
// Search for attention mechanism info
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "tools/call", 
  "params": {
    "name": "search_memories",
    "arguments": {
      "query": "attention mechanism transformer",
      "top_k": 5,
      "cube_ids": ["ai-research", "project-alpha"]
    }
  }
}

// Chat about findings
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "tools/call",
  "params": {
    "name": "chat",
    "arguments": {
      "query": "How did implementing attention mechanisms improve our project performance?",
      "max_tokens": 300,
      "top_k": 3
    }
  }
}
```

---

### Example 2: Personal Learning Journal

Track and query personal learning across multiple topics.

#### Setup Learning Cubes

```json
// Programming knowledge
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "register_cube", 
    "arguments": {
      "cube_id": "programming-knowledge",
      "name": "Programming Concepts",
      "description": "Programming languages, frameworks, and best practices"
    }
  }
}

// Career development
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "register_cube",
    "arguments": {
      "cube_id": "career-development",
      "name": "Career Growth",
      "description": "Skills, experiences, and career insights"
    }
  }
}
```

#### Daily Learning Entries

```json
// Learning about Go programming
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "add_memory",
    "arguments": {
      "memory_content": "Learned about Go goroutines today. They're lightweight threads managed by the Go runtime. Key advantages: easy concurrency with 'go' keyword, channels for communication, built-in scheduler. Much simpler than traditional threading in other languages.",
      "mem_cube_id": "programming-knowledge",
      "metadata": {
        "language": "go",
        "topic": "concurrency",
        "difficulty": "intermediate",
        "date": "2024-01-15",
        "source": "official_docs"
      }
    }
  }
}

// Career insight
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "add_memory",
    "arguments": {
      "memory_content": "Had a great mentoring session with Sarah. Key takeaway: focus on becoming a T-shaped professional - deep expertise in one area (AI/ML for me) but broad knowledge across related fields (distributed systems, product management, etc.).",
      "mem_cube_id": "career-development", 
      "metadata": {
        "type": "mentoring",
        "mentor": "Sarah",
        "key_concept": "T-shaped_professional",
        "date": "2024-01-15"
      }
    }
  }
}
```

#### Weekly Review Queries

```json
// What did I learn about programming this week?
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "tools/call",
  "params": {
    "name": "chat",
    "arguments": {
      "query": "What programming concepts did I learn this week? Summarize the key insights.",
      "cube_ids": ["programming-knowledge"],
      "max_tokens": 400
    }
  }
}

// Review career development progress
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "tools/call",
  "params": {
    "name": "search_memories",
    "arguments": {
      "query": "career growth T-shaped professional",
      "cube_ids": ["career-development"],
      "top_k": 3
    }
  }
}
```

---

## 🎯 Advanced Workflows

### Example 3: Multi-Project Knowledge Management

Managing knowledge across multiple projects with shared concepts.

#### Project Structure Setup

```json
// Shared knowledge base
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "register_cube",
    "arguments": {
      "cube_id": "shared-knowledge",
      "name": "Shared Technical Knowledge",
      "description": "Common patterns, libraries, and best practices"
    }
  }
}

// Project-specific cubes
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "register_cube",
    "arguments": {
      "cube_id": "ecommerce-platform",
      "name": "E-commerce Platform",
      "description": "Online shopping platform development"
    }
  }
}

{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "register_cube",
    "arguments": {
      "cube_id": "data-pipeline",
      "name": "Data Processing Pipeline",
      "description": "ETL pipeline for analytics"
    }
  }
}
```

#### Adding Cross-Referenced Knowledge

```json
// Shared architectural pattern
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "add_memory",
    "arguments": {
      "memory_content": "Event-driven architecture pattern: Services communicate through events rather than direct API calls. Benefits: loose coupling, scalability, resilience. Use cases: order processing, user notifications, audit logging. Tools: Apache Kafka, AWS EventBridge, Redis Streams.",
      "mem_cube_id": "shared-knowledge",
      "metadata": {
        "pattern": "event-driven-architecture",
        "type": "architectural-pattern",
        "complexity": "intermediate",
        "applicable_projects": ["ecommerce-platform", "data-pipeline"]
      }
    }
  }
}

// E-commerce specific implementation
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "tools/call",
  "params": {
    "name": "add_memory",
    "arguments": {
      "memory_content": "Implemented event-driven order processing: OrderCreated → PaymentProcessed → InventoryReserved → ShippingScheduled → OrderCompleted. Using Kafka with 3 partitions per topic for scalability. Average processing time: 150ms per order.",
      "mem_cube_id": "ecommerce-platform",
      "metadata": {
        "feature": "order-processing",
        "technology": "kafka",
        "performance": "150ms",
        "references": ["shared-knowledge/event-driven-architecture"]
      }
    }
  }
}
```

#### Cross-Project Queries

```json
// Find patterns applicable to current project
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "tools/call",
  "params": {
    "name": "context_search",
    "arguments": {
      "query": "scalable architecture patterns for real-time data processing",
      "cube_ids": ["shared-knowledge", "data-pipeline"],
      "top_k": 5
    }
  }
}

// Compare implementations across projects
{
  "jsonrpc": "2.0",
  "id": 7,
  "method": "tools/call",
  "params": {
    "name": "chat",
    "arguments": {
      "query": "How are we using event-driven patterns across different projects? What are the performance characteristics?",
      "max_tokens": 500,
      "top_k": 5
    }
  }
}
```

---

### Example 4: Research Paper Analysis Workflow

Systematically analyze and cross-reference research papers.

#### Paper Processing Pipeline

```json
// Create research cube
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "register_cube",
    "arguments": {
      "cube_id": "ml-papers-2024",
      "name": "ML Papers 2024",
      "description": "Machine learning research papers from 2024"
    }
  }
}

// Add paper with structured metadata
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "add_memory",
    "arguments": {
      "memory_content": "PAPER ANALYSIS:\n\nTitle: 'Scaling Language Models with Mixture of Experts'\nAuthors: Chen et al. (2024)\n\nKey Contributions:\n1. Novel sparse MoE architecture reducing computational cost by 60%\n2. Dynamic expert routing based on input complexity\n3. New training methodology for expert specialization\n\nMethodology:\n- Compared against dense Transformer baselines\n- Evaluated on 5 NLP benchmarks (GLUE, SuperGLUE, etc.)\n- Scaling experiments from 1B to 100B parameters\n\nResults:\n- 60% reduction in FLOPs with minimal performance degradation\n- Better performance on specialized tasks (code, math)\n- Improved inference speed: 2.3x faster than dense models\n\nLimitations:\n- Memory requirements still high due to expert parameters\n- Training complexity increased\n- Expert load balancing challenges\n\nRelevance to our work: Could apply MoE techniques to our domain-specific models",
      "mem_cube_id": "ml-papers-2024",
      "metadata": {
        "title": "Scaling Language Models with Mixture of Experts",
        "authors": ["Chen", "Liu", "Wang"],
        "year": 2024,
        "venue": "NeurIPS",
        "keywords": ["mixture-of-experts", "language-models", "efficiency"],
        "performance_gain": "60% FLOP reduction",
        "relevance_score": 9,
        "status": "analyzed"
      }
    }
  }
}
```

#### Literature Review Queries

```json
// Find papers on efficiency techniques
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "search_memories",
    "arguments": {
      "query": "model efficiency techniques FLOP reduction sparse",
      "cube_ids": ["ml-papers-2024"],
      "top_k": 10
    }
  }
}

// Synthesize findings
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "chat",
    "arguments": {
      "query": "What are the main approaches to improving language model efficiency according to recent papers? Compare their trade-offs.",
      "max_tokens": 600,
      "top_k": 8
    }
  }
}

// Track research trends
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "tools/call",
  "params": {
    "name": "context_search",
    "arguments": {
      "query": "mixture of experts routing expert specialization",
      "top_k": 5
    }
  }
}
```

---

## 🔧 Integration Examples

### Example 5: Claude Desktop Integration

Complete workflow for using MemGOS through Claude Desktop.

#### Configuration
```json
// ~/.config/claude/claude_desktop_config.json
{
  "mcpServers": {
    "memgos": {
      "command": "/usr/local/bin/memgos-mcp",
      "args": [
        "--api-url", "http://localhost:8080",
        "--user", "claude-user",
        "--session", "claude-desktop-session"
      ],
      "env": {
        "MEMGOS_API_TOKEN": "your-secure-api-token"
      }
    }
  }
}
```

#### Natural Language Interactions

```
Human: I want to start tracking my learning about distributed systems. Can you help me set up a knowledge base?

Claude: I'll help you create a knowledge base for distributed systems learning. Let me set up a memory cube for you.

[Claude uses register_cube tool]

I've created a "distributed-systems" memory cube for you. Now you can start adding your learning notes, and I'll help you organize and recall them later.

---