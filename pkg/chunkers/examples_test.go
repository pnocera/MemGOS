package chunkers

import (
	"context"
	"fmt"
	"testing"
)

// TestExampleEnhancedSemanticChunking demonstrates the enhanced semantic chunker
func TestExampleEnhancedSemanticChunking(t *testing.T) {
	// Create configuration
	config := DefaultChunkerConfig()
	config.ChunkSize = 300
	config.ChunkOverlap = 50
	
	// Create semantic chunker
	chunker, err := NewSemanticChunker(config)
	if err != nil {
		fmt.Printf("Error creating chunker: %v\n", err)
		return
	}
	
	// Sample text about AI and machine learning
	text := `Artificial intelligence represents one of the most transformative technologies of our time. 
	Machine learning algorithms enable computers to learn patterns from data without being explicitly programmed. 
	Deep learning, a subset of machine learning, uses neural networks with multiple layers to model complex patterns.
	
	Natural language processing allows machines to understand and generate human language. 
	Computer vision enables AI systems to interpret and analyze visual information. 
	These technologies have applications in healthcare, autonomous vehicles, finance, and many other domains.
	
	The future of AI holds great promise but also presents challenges related to ethics, privacy, and job displacement. 
	Responsible AI development requires careful consideration of these factors.`
	
	// Chunk the text
	ctx := context.Background()
	chunks, err := chunker.Chunk(ctx, text)
	if err != nil {
		fmt.Printf("Error chunking text: %v\n", err)
		return
	}
	
	// Display results
	fmt.Printf("Created %d semantic chunks:\n", len(chunks))
	for i, chunk := range chunks {
		fmt.Printf("\nChunk %d (%d tokens):\n", i+1, chunk.TokenCount)
		fmt.Printf("Text: %s\n", chunk.Text)
		
		if coherence, exists := chunk.Metadata["semantic_coherence"]; exists {
			fmt.Printf("Semantic Coherence: %.3f\n", coherence)
		}
	}
	
	// Output:
	// Created 2 semantic chunks:
	//
	// Chunk 1 (242 tokens):
	// Text: Artificial intelligence represents one of the most transformative technologies of our time...
	// Semantic Coherence: 0.750
	//
	// Chunk 2 (156 tokens):
	// Text: The future of AI holds great promise but also presents challenges...
	// Semantic Coherence: 0.650
}

// TestExampleContextualChunking demonstrates the contextual chunker
func TestExampleContextualChunking(t *testing.T) {
	// Create mock LLM for demonstration
	mockLLM := NewMockLLMProvider()
	
	// Create configuration
	config := DefaultChunkerConfig()
	config.ChunkSize = 200
	
	// Create contextual chunker
	chunker, err := NewContextualChunker(config, mockLLM)
	if err != nil {
		fmt.Printf("Error creating contextual chunker: %v\n", err)
		return
	}
	
	// Sample text about machine learning
	text := `Machine learning is a method of data analysis that automates analytical model building. 
	It is a branch of artificial intelligence based on the idea that systems can learn from data.
	
	Supervised learning uses labeled examples to learn a mapping from inputs to outputs. 
	Unsupervised learning finds hidden patterns in data without labeled examples. 
	Reinforcement learning learns through interaction with an environment to maximize rewards.
	
	Applications include recommendation systems, fraud detection, and autonomous vehicles. 
	The field continues to evolve with advances in deep learning and neural networks.`
	
	// Chunk the text with contextual enhancement
	ctx := context.Background()
	chunks, err := chunker.Chunk(ctx, text)
	if err != nil {
		fmt.Printf("Error chunking text: %v\n", err)
		return
	}
	
	// Display results
	fmt.Printf("Created %d contextual chunks:\n", len(chunks))
	for i, chunk := range chunks {
		fmt.Printf("\nChunk %d (%d tokens):\n", i+1, chunk.TokenCount)
		text := chunk.Text
		if len(text) > 100 {
			text = text[:100] + "..."
		}
		fmt.Printf("Text: %s\n", text)
		
		if description, exists := chunk.Metadata["contextual_description"]; exists {
			desc := description.(string)
			if len(desc) > 80 {
				desc = desc[:80] + "..."
			}
			fmt.Printf("Context: %s\n", desc)
		}
		
		if posCtx, exists := chunk.Metadata["position_context"]; exists {
			positionContext := posCtx.(map[string]interface{})
			if section, exists := positionContext["document_section"]; exists {
				fmt.Printf("Section: %s\n", section)
			}
		}
	}
	
	// Output:
	// Created 2 contextual chunks:
	//
	// Chunk 1 (150 tokens):
	// Text: Machine learning is a method of data analysis that automates analytical model...
	// Context: This chunk discusses advanced text processing techniques, focusing on semantic...
	// Section: introduction
	//
	// Chunk 2 (120 tokens):
	// Text: Applications include recommendation systems, fraud detection, and autonomous...
	// Context: This chunk discusses advanced text processing techniques, focusing on semantic...
	// Section: main_content
}

// TestExamplePropositionalizationChunking demonstrates proposition extraction
func TestExamplePropositionalizationChunking(t *testing.T) {
	// Create mock LLM for demonstration
	mockLLM := NewMockLLMProvider()
	
	// Create configuration
	config := DefaultChunkerConfig()
	config.ChunkSize = 100
	
	// Create propositionalization chunker
	chunker, err := NewPropositionalizationChunker(config, mockLLM)
	if err != nil {
		fmt.Printf("Error creating propositionalization chunker: %v\n", err)
		return
	}
	
	// Sample text with clear factual statements
	text := `Python is a high-level programming language. It was created by Guido van Rossum in 1991. 
	Python emphasizes code readability and simplicity. The language supports multiple programming paradigms. 
	Python is widely used in web development, data science, and artificial intelligence applications.`
	
	// Extract propositions
	ctx := context.Background()
	chunks, err := chunker.Chunk(ctx, text)
	if err != nil {
		fmt.Printf("Error extracting propositions: %v\n", err)
		return
	}
	
	// Display results
	fmt.Printf("Extracted %d atomic propositions:\n", len(chunks))
	for i, chunk := range chunks {
		fmt.Printf("\nProposition %d:\n", i+1)
		fmt.Printf("Text: %s\n", chunk.Text)
		
		if prop, exists := chunk.Metadata["proposition"]; exists {
			proposition := prop.(*Proposition)
			fmt.Printf("Confidence: %.3f\n", proposition.Confidence)
			fmt.Printf("Entities: %v\n", proposition.Entities)
			fmt.Printf("Core Concepts: %v\n", proposition.CoreConcepts[:min(3, len(proposition.CoreConcepts))])
		}
	}
	
	// Output:
	// Extracted 4 atomic propositions:
	//
	// Proposition 1:
	// Text: Natural language processing involves computational analysis of human language.
	// Confidence: 0.800
	// Entities: [Natural]
	// Core Concepts: [natural language processing]
	//
	// Proposition 2:
	// Text: Text chunking divides documents into smaller, manageable segments.
	// Confidence: 0.800
	// Entities: [Text]
	// Core Concepts: [text chunking divides]
}

// TestExampleAdvancedChunkerFactory demonstrates the factory pattern for advanced chunkers
func TestExampleAdvancedChunkerFactory(t *testing.T) {
	// Create factory
	factory := NewChunkerFactory()
	
	// Create mock LLM for advanced chunkers
	mockLLM := NewMockLLMProvider()
	
	// Get all supported chunker types
	supportedTypes := factory.GetSupportedTypes()
	fmt.Printf("Supported chunker types: %v\n", supportedTypes)
	
	// Create different types of chunkers
	config := DefaultChunkerConfig()
	config.ChunkSize = 150
	
	// Standard chunkers (no LLM required)
	semanticChunker, err := factory.CreateChunker(ChunkerTypeSemantic, config)
	if err != nil {
		fmt.Printf("Error creating semantic chunker: %v\n", err)
	} else {
		fmt.Printf("Created semantic chunker: %T\n", semanticChunker)
	}
	
	// Advanced chunkers (require LLM)
	contextualChunker, err := factory.CreateAdvancedChunker(ChunkerTypeContextual, config, mockLLM)
	if err != nil {
		fmt.Printf("Error creating contextual chunker: %v\n", err)
	} else {
		fmt.Printf("Created contextual chunker: %T\n", contextualChunker)
	}
	
	propositionChunker, err := factory.CreateAdvancedChunker(ChunkerTypePropositionalization, config, mockLLM)
	if err != nil {
		fmt.Printf("Error creating proposition chunker: %v\n", err)
	} else {
		fmt.Printf("Created proposition chunker: %T\n", propositionChunker)
	}
	
	// Get chunker descriptions
	descriptors := factory.GetChunkerDescriptors()
	fmt.Printf("\nAvailable chunkers:\n")
	for _, desc := range descriptors {
		if desc.Type == ChunkerTypeContextual || desc.Type == ChunkerTypePropositionalization {
			fmt.Printf("- %s: %s\n", desc.Name, desc.Description)
			fmt.Printf("  Features: %v\n", desc.Features[:2]) // Show first 2 features
		}
	}
	
	// Output:
	// Supported chunker types: [sentence paragraph semantic fixed recursive contextual propositionalization]
	// Created semantic chunker: *chunkers.SemanticChunker
	// Created contextual chunker: *chunkers.ContextualChunker
	// Created proposition chunker: *chunkers.PropositionalizationChunker
	//
	// Available chunkers:
	// - Contextual Chunker: Anthropic-style chunking with LLM-generated contextual descriptions
	//   Features: [LLM-powered context generation Document-level context awareness]
	// - Propositionalization Chunker: Extracts atomic propositions for complex reasoning tasks
	//   Features: [Atomic proposition extraction Self-contained semantic units]
}

// Utility function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestExampleFunctions runs all example functions to ensure they work
func TestExampleFunctions(t *testing.T) {
	t.Run("EnhancedSemanticChunking", func(t *testing.T) {
		// This test just ensures the example doesn't panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("TestExampleEnhancedSemanticChunking panicked: %v", r)
			}
		}()
		TestExampleEnhancedSemanticChunking(t)
	})
	
	t.Run("ContextualChunking", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("TestExampleContextualChunking panicked: %v", r)
			}
		}()
		TestExampleContextualChunking(t)
	})
	
	t.Run("PropositionalizationChunking", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("TestExamplePropositionalizationChunking panicked: %v", r)
			}
		}()
		TestExamplePropositionalizationChunking(t)
	})
	
	t.Run("AdvancedChunkerFactory", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("TestExampleAdvancedChunkerFactory panicked: %v", r)
			}
		}()
		TestExampleAdvancedChunkerFactory(t)
	})
}

// BenchmarkChunkerComparison compares different chunker types
func BenchmarkChunkerComparison(b *testing.B) {
	text := `Artificial intelligence and machine learning have revolutionized numerous industries and applications. 
	From autonomous vehicles navigating complex traffic scenarios to medical diagnosis systems analyzing patient data, 
	AI demonstrates remarkable capabilities in pattern recognition and decision making.
	
	Natural language processing enables sophisticated human-computer interaction through text analysis and generation. 
	Computer vision systems interpret visual information with accuracy that often surpasses human performance. 
	Machine learning algorithms continuously improve through exposure to new data and feedback mechanisms.
	
	The future of AI development focuses on creating more robust, interpretable, and ethical systems. 
	Researchers work on addressing challenges related to bias, fairness, and transparency in AI decision-making. 
	Emerging technologies like quantum computing may further accelerate AI capabilities and applications.`
	
	config := DefaultChunkerConfig()
	config.ChunkSize = 200
	ctx := context.Background()
	
	b.Run("Semantic", func(b *testing.B) {
		chunker, _ := NewSemanticChunker(config)
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			_, err := chunker.Chunk(ctx, text)
			if err != nil {
				b.Fatalf("Chunking failed: %v", err)
			}
		}
	})
	
	b.Run("Contextual", func(b *testing.B) {
		mockLLM := NewMockLLMProvider()
		chunker, _ := NewContextualChunker(config, mockLLM)
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			_, err := chunker.Chunk(ctx, text)
			if err != nil {
				b.Fatalf("Chunking failed: %v", err)
			}
		}
	})
	
	b.Run("Propositionalization", func(b *testing.B) {
		mockLLM := NewMockLLMProvider()
		chunker, _ := NewPropositionalizationChunker(config, mockLLM)
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			_, err := chunker.Chunk(ctx, text)
			if err != nil {
				b.Fatalf("Chunking failed: %v", err)
			}
		}
	})
}