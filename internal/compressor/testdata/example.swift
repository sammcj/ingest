// This is a Swift comment
/* This is a multi-line
   Swift comment */

// Import statements
import Foundation

// Constants and variables
let maxAge = 120
var count = 0

// Struct definition
struct Address {
    let street: String
    let city: String
    let zipCode: String

    // Computed property
    var fullAddress: String {
        return "\(street), \(city) \(zipCode)"
    }
}

// Class definition
class Person {
    // Properties
    var name: String
    var age: Int
    var address: Address?

    // Static property
    static var count = 0

    // Initializer
    init(name: String, age: Int) {
        self.name = name
        self.age = age
        Person.count += 1
    }

    // Convenience initializer
    convenience init(name: String) {
        self.init(name: name, age: 0)
    }

    // Deinitializer
    deinit {
        Person.count -= 1
    }

    // Method with parameters
    func setAddress(street: String, city: String, zipCode: String) {
        self.address = Address(street: street, city: city, zipCode: zipCode)
    }

    // Method with return value
    func getInfo() -> String {
        if let address = address {
            return "\(name), age \(age), lives at \(address.fullAddress)"
        } else {
            return "\(name), age \(age)"
        }
    }

    // Method with parameters and return value
    func isOlderThan(_ other: Person) -> Bool {
        return self.age > other.age
    }

    // Static method
    static func getCount() -> Int {
        return count
    }
}

// Extension
extension Person {
    // Additional method in extension
    func celebrateBirthday() {
        age += 1
        print("\(name) is now \(age) years old!")
    }
}

// Protocol definition
protocol Printable {
    func print()
}

// Protocol extension
extension Printable {
    // Default implementation
    func printWithPrefix(_ prefix: String) {
        Swift.print("\(prefix): \(self)")
    }
}

// Protocol conformance
extension Person: Printable {
    func print() {
        Swift.print(getInfo())
    }
}

// Enum definition
enum Status {
    case active
    case inactive
    case pending(String)
    case error(code: Int, message: String)

    // Method in enum
    func description() -> String {
        switch self {
        case .active:
            return "Active"
        case .inactive:
            return "Inactive"
        case .pending(let reason):
            return "Pending: \(reason)"
        case .error(let code, let message):
            return "Error \(code): \(message)"
        }
    }
}

// Generic function
func process<T: Printable>(_ item: T) {
    item.print()
}

// Closure
let greet = { (name: String) -> String in
    return "Hello, \(name)!"
}

// Main function equivalent
func main() {
    // Create objects
    let person1 = Person(name: "John Doe", age: 30)
    person1.setAddress(street: "123 Main St", city: "Anytown", zipCode: "12345")

    let person2 = Person(name: "Jane Smith", age: 25)
    person2.setAddress(street: "456 Oak Ave", city: "Somewhere", zipCode: "67890")

    // Use methods
    print(person1.getInfo())
    print("Total persons: \(Person.getCount())")

    // Conditional statement
    if person1.isOlderThan(person2) {
        print("\(person1.name) is older than \(person2.name)")
    } else {
        print("\(person2.name) is older than \(person1.name)")
    }

    // Collections
    var people = [person1, person2]

    // Higher-order functions
    let averageAge = people.reduce(0) { $0 + $1.age } / people.count
    print("Average age: \(averageAge)")

    // Closure usage
    people.forEach { person in
        print(person.name)
    }

    // Error handling
    do {
        let data = try Data(contentsOf: URL(fileURLWithPath: "/nonexistent.txt"))
        print("File size: \(data.count)")
    } catch {
        print("Error reading file: \(error)")
    }

    // Pattern matching
    let status = Status.pending("Awaiting approval")
    switch status {
    case .active:
        print("Active")
    case .inactive:
        print("Inactive")
    case .pending(let reason):
        print("Pending: \(reason)")
    case .error(let code, let message):
        print("Error \(code): \(message)")
    }
}

// Call main function
main()
