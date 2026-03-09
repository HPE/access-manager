

## Example: NGFW Using AWS Tags with Ambiguous Rule Processing

---

### **Scenario**
A company is using an NGFW integrated with AWS. The NGFW defines **address book objects** based on **AWS resource tags**. These objects are used in firewall rules to control traffic flow between different application tiers (e.g., web, application, and database). 

The rules are processed in order, and overlapping resources in address book objects create **unexpected results**.

---

### **1. AWS Tag-Based Address Book Objects**
The NGFW dynamically retrieves AWS resource tags to define address book objects:

| Address Book Object | AWS Tag Key       | AWS Tag Value     | Resources Covered                       |
|--------------------|----------------|-----------------|-------------------------------|
| Web-Tier                     | `Environment`    | `prod`            | All resources tagged with `prod`.      |
| App-Tier                      | `Role`                 | `app`             | All resources tagged with `Role=app`.  |
| Database-Tier             | `Role`                 | `db`              | All resources tagged with `Role=db`.   |
| Shared-Resources     | `Shared`             | `true`            | All shared resources.                  |


---

### **2. NGFW Rules**

| Rule | Name                     | Source Address Object | Destination Address Object | Protocol | Port  | Action | Notes |
|------|--------------------------|-----------------------|----------------------------|----------|-------|--------|-------|
| 1    | Block-Shared-to-App      | Shared-Resources      | App-Tier  | Any      | Any   | Deny   | Blocks shared resources from accessing the app tier. |
| 2    | Allow-Web-to-App         | Web-Tier              | App-Tier                   | TCP      | 8080  | Allow  | Allows web tier to access app tier over HTTP. |
| 3    | Allow-App-to-Database    | App-Tier              | Database-Tier              | TCP      | 3306  | Allow  | Allows app tier to access database tier over MySQL. |
| 4    | Allow-Shared-to-Database | Shared-Resources      | Database-Tier              | TCP      | 3306  | Allow  | Allows shared resources to access database tier over MySQL. |
| 5    | Block-All                | Any                   | Any                        | Any      | Any   | Deny   | Default deny all rule. |

---

### **3. Overlap in Address Book Objects**

- **Issue:** Resources tagged with `Environment=prod` (covered by `Web-Tier`) **may also have additional tags**, such as `Role=app` or `Role=db`.
  - Example: A resource tagged as:
    - `Environment=prod`
    - `Role=app`
  - This resource belongs to **both the Web-Tier and App-Tier** address book objects, creating ambiguity in rule processing.

- **Unexpected Results:**
  - If the above resource tries to communicate with another resource in the `App-Tier`, **Rule 1 (Block-Shared-to-App)** might incorrectly block the traffic, depending on how the NGFW processes overlapping address book objects.

---

### **4. Ambiguous Rule Processing**

#### Example 1: Resource in Web-Tier and App-Tier
- **Resource:** Tagged with `Environment=prod` and `Role=app`.
- **Scenario:** This resource attempts to access another resource in the `App-Tier`.
- **Expected Behavior:** Rule 2 (`Allow-Web-to-App`) should allow the traffic.
- **Actual Behavior:**
  - If the NGFW processes Rule 1 (`Block-Shared-to-App`) first, it may incorrectly block the traffic, because the resource is also part of `Shared-Resources`.

#### Example 2: Resource in Shared-Resources and Database-Tier
- **Resource:** Tagged with `Shared=true` and `Role=db`.
- **Scenario:** This resource attempts to access another resource in the `Database-Tier` over MySQL.
- **Expected Behavior:** Rule 4 (`Allow-Shared-to-Database`) should allow the traffic.
- **Actual Behavior:**
  - If Rule 1 (`Block-Shared-to-App`) is processed first, the traffic might be blocked, because the NGFW does not distinguish that the resource is also part of `Database-Tier`.

---

### **5. Recommendations to Avoid Unexpected Results**

1. **Avoid Overlapping Tags:**
   - Ensure that AWS tags used for defining address book objects do not overlap across different roles or tiers.

2. **Prioritize Specific Rules:**
   - Place rules with more specific matches (e.g., `Allow-Web-to-App`) above more general rules (e.g., `Block-Shared-to-App`).

3. **Use Explicit Conditions:**
   - Instead of relying solely on tags, use additional criteria (e.g., IP ranges or security groups) to define address book objects.

4. **Enable Logging and Monitoring:**
   - Monitor traffic logs to identify and resolve rule processing ambiguities in real-time.

5. **Test Rules in a Staging Environment:**
   - Simulate traffic scenarios before deploying rules in production to ensure the expected behavior.

---

### **6. Fixed Rule Order to Resolve Ambiguities**

By reordering the rules and prioritizing specific conditions, the ambiguities can be resolved:

| Priority | Rule Name              | Action  |
|----------|------------------------|---------|
| 1        | Allow-Web-to-App       | Allow   |
| 2        | Allow-App-to-Database  | Allow   |
| 3        | Allow-Shared-to-Database | Allow   |
| 4        | Block-Shared-to-App    | Deny    |
| 5        | Block-All              | Deny    |

This ensures that **specific allow rules are processed first**, avoiding unintended blocks caused by overlapping address book objects.

---

### **Key Takeaways**
1. Overlapping AWS tags can create ambiguities in NGFW rule processing when defining address book objects.
2. Rule prioritization and explicit conditions are critical to avoid unintended behavior.
3. Testing and monitoring are essential to ensure that rules achieve the desired outcomes.

Let me know if you'd like additional examples or further clarification!