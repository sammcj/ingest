// This is a Java comment
/* This is a multi-line
   Java comment */

// Package declaration
package com.example.demo;

// Import statements
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.stream.Collectors;

/**
 * This is a Javadoc comment for the Person class
 * @author Example Author
 */
public class Person {
    // Instance variables
    private String name;
    private int age;
    private String address;

    // Static variable
    private static int count = 0;

    // Constants
    public static final int MAX_AGE = 120;

    // Constructor
    public Person(String name, int age) {
        this.name = name;
        this.age = age;
        this.address = null;
        count++;
    }

    // Getter methods
    public String getName() {
        return name;
    }

    public int getAge() {
        return age;
    }

    public String getAddress() {
        return address;
    }

    // Setter methods
    public void setName(String name) {
        this.name = name;
    }

    public void setAge(int age) {
        if (age > MAX_AGE) {
            throw new IllegalArgumentException("Age cannot be greater than " + MAX_AGE);
        }
        this.age = age;
    }

    public void setAddress(String address) {
        this.address = address;
    }

    // Static method
    public static int getCount() {
        return count;
    }

    // Method with return value
    public String getInfo() {
        if (address != null) {
            return name + ", age " + age + ", lives at " + address;
        } else {
            return name + ", age " + age;
        }
    }

    // Method with parameters
    public boolean isOlderThan(Person other) {
        return this.age > other.age;
    }

    // Override toString method
    @Override
    public String toString() {
        return getInfo();
    }

    // Inner class
    public class Address {
        private String street;
        private String city;
        private String zipCode;

        public Address(String street, String city, String zipCode) {
            this.street = street;
            this.city = city;
            this.zipCode = zipCode;
        }

        public String getFullAddress() {
            return street + ", " + city + " " + zipCode;
        }
    }
}

// Interface definition
interface Printable {
    void print();

    // Default method
    default void printWithPrefix(String prefix) {
        System.out.println(prefix + ": " + toString());
    }
}

// Enum definition
enum Status {
    ACTIVE("Active"),
    INACTIVE("Inactive"),
    PENDING("Pending");

    private final String label;

    Status(String label) {
        this.label = label;
    }

    public String getLabel() {
        return label;
    }
}

// Main class
public class Main {
    public static void main(String[] args) {
        // Create objects
        Person person1 = new Person("John Doe", 30);
        person1.setAddress("123 Main St");

        Person person2 = new Person("Jane Smith", 25);
        person2.setAddress("456 Oak Ave");

        // Use methods
        System.out.println(person1.getInfo());
        System.out.println("Total persons: " + Person.getCount());

        // Conditional statement
        if (person1.isOlderThan(person2)) {
            System.out.println(person1.getName() + " is older than " + person2.getName());
        } else {
            System.out.println(person2.getName() + " is older than " + person1.getName());
        }

        // Collections
        List<Person> people = new ArrayList<>();
        people.add(person1);
        people.add(person2);

        // Stream API
        double averageAge = people.stream()
                .mapToInt(Person::getAge)
                .average()
                .orElse(0);

        System.out.println("Average age: " + averageAge);

        // Lambda expression
        people.forEach(p -> System.out.println(p.getName()));

        // Try-catch block
        try {
            Person invalidPerson = new Person("Invalid", 150);
        } catch (IllegalArgumentException e) {
            System.out.println("Error: " + e.getMessage());
        } finally {
            System.out.println("Finished processing");
        }
    }
}
