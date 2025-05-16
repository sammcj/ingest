// This is a Rust comment
/* This is a multi-line
   Rust comment */

// Import statements
use std::collections::HashMap;
use std::io::{self, Read, Write};
use std::sync::{Arc, Mutex};

// Constants and statics
const MAX_SIZE: usize = 100;
static GLOBAL_COUNTER: std::sync::atomic::AtomicUsize = std::sync::atomic::AtomicUsize::new(0);

// Struct definition
pub struct Person {
    name: String,
    age: u32,
    address: Option<String>,
}

// Implementation block
impl Person {
    // Constructor
    pub fn new(name: &str, age: u32) -> Self {
        Person {
            name: name.to_string(),
            age,
            address: None,
        }
    }

    // Method with parameters
    pub fn set_address(&mut self, address: String) {
        self.address = Some(address);
    }

    // Method with return value
    pub fn get_info(&self) -> String {
        match &self.address {
            Some(addr) => format!("{}, age {}, lives at {}", self.name, self.age, addr),
            None => format!("{}, age {}", self.name, self.age),
        }
    }
}

// Trait definition
pub trait Printable {
    fn print(&self);

    // Default implementation
    fn print_debug(&self) {
        println!("Debug print");
    }
}

// Trait implementation
impl Printable for Person {
    fn print(&self) {
        println!("{}", self.get_info());
    }
}

// Enum definition
pub enum Status {
    Active,
    Inactive,
    Pending(String),
    Error { code: u32, message: String },
}

// Function with generic type
pub fn process<T: Printable>(item: &T) {
    item.print();
}

// Main function
fn main() {
    // Variable declaration
    let mut person = Person::new("John Doe", 30);
    person.set_address("123 Main St".to_string());

    // Function call
    process(&person);

    // Pattern matching
    let status = Status::Pending("Awaiting approval".to_string());
    match status {
        Status::Active => println!("Active"),
        Status::Inactive => println!("Inactive"),
        Status::Pending(reason) => println!("Pending: {}", reason),
        Status::Error { code, message } => println!("Error {}: {}", code, message),
    }

    // Closure
    let add = |a: i32, b: i32| a + b;
    println!("5 + 3 = {}", add(5, 3));

    // Error handling
    let result = std::fs::read_to_string("nonexistent.txt");
    match result {
        Ok(content) => println!("File content: {}", content),
        Err(error) => println!("Error reading file: {}", error),
    }
}
