package hello

object Hello

fun main() {
    println("hello from ${Hello::class.java.`package`.implementationVersion}")
}
