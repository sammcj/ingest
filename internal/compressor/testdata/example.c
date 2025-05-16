#include <stdio.h>
#include <stdlib.h>
#include <string.h>

// Define a struct
struct Person {
    char name[50];
    int age;
    float height;
};

// Define an enum
enum Color {
    RED,
    GREEN,
    BLUE,
    YELLOW
};

// Define a union
union Data {
    int i;
    float f;
    char str[20];
};

// Function prototype
void greet(const char* name);
int calculate_sum(int a, int b);

// Typedef example
typedef unsigned long int UINT32;

/**
 * Main function
 */
int main(int argc, char *argv[]) {
    // Variable declarations
    struct Person person;
    enum Color favorite_color = BLUE;
    union Data data;
    UINT32 big_number = 123456789UL;

    // Initialize struct
    strcpy(person.name, "John");
    person.age = 30;
    person.height = 1.75;

    // Print information
    printf("Name: %s\n", person.name);
    printf("Age: %d\n", person.age);
    printf("Height: %.2f\n", person.height);

    // Call functions
    greet(person.name);
    printf("Sum: %d\n", calculate_sum(5, 7));

    // Use union
    data.i = 10;
    printf("data.i: %d\n", data.i);
    data.f = 220.5;
    printf("data.f: %.2f\n", data.f);
    strcpy(data.str, "C Programming");
    printf("data.str: %s\n", data.str);

    return 0;
}

/**
 * Greet function implementation
 */
void greet(const char* name) {
    printf("Hello, %s!\n", name);

    if (strlen(name) > 5) {
        printf("You have a long name!\n");
    } else {
        printf("You have a short name!\n");
    }
}

/**
 * Calculate sum function implementation
 */
int calculate_sum(int a, int b) {
    int result = a + b;
    return result;
}
