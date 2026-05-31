# Go-Stop — Product Requirements

## Overview

A lightweight local car sharing platform positioned between hitchhiking and BlaBlaCar. The platform connects drivers offering one-time local rides with people seeking a lift, via a simple web-based notice board with push notifications.

## Concept

- Drivers **propose** rides to a destination
- Searchers **browse** available rides or post a waiting request
- Both parties get **notified** when a match is found
- Contact is made directly via phone number — no in-app messaging

## Delivery

- **Website** (no native mobile app)
- **Progressive Web App** with mobile push notifications
- Hosted entirely on **Scalingo** (Go app + PostgreSQL add-on)

---

## User Requirements

### Identity
- No user accounts required
- Users enter a **name** and **phone number** when posting a ride or request
- Phone number serves as lightweight authentication for deletions
- No phone verification — frictionless entry
- Phone number is displayed on ride/request cards so parties can contact each other directly

### Locations
- No predefined destination list
- Destinations are **user-generated** and self-building (populated from past posts)
- Autocomplete draws from existing origins and destinations
- Suitable for both urban and rural contexts (e.g. next village, train station)

### Rides
- **One-time trips only** — no recurring schedules
- Each ride has an origin, destination, date, departure time, and flexibility window
- Rides expire automatically after their timeframe passes
- Drivers can manually delete their own ride (authenticated by phone number)

### Requests
- Searchers can post a waiting request if no matching ride exists
- Requests have the same fields as rides (origin, destination, date, departure time, flexibility)
- Requests expire automatically
- Searchers can manually delete their own request (authenticated by phone number)

### Matching
- When a ride is posted → match against existing requests → notify matching searchers
- When a request is posted → match against existing rides → notify matching drivers
- Matching considers origin, destination, date, and overlapping flexibility windows

### Notifications
- **Bidirectional** push notifications via Web Push API
- Drivers notified when a searcher requests their route
- Searchers notified when a matching ride is posted
- Notifications include: contact name, phone number, origin, destination, departure time, and a deep link to the ride/request

### Cost Sharing
- Out of scope — no payment functionality

---

## Non-Functional Requirements

- **Simplicity first** — anyone should understand how it works on first visit
- **Low friction** — minimal steps to post or find a ride
- **Privacy conscious** — no personal data stored beyond what is needed for the ride/request lifespan
- **Transient data** — rides, requests, and associated phone numbers are deleted on expiry
- No mobile app — website with PWA capabilities is sufficient

---

## User Flows

### Driver Flow
1. Land on homepage
2. Tap **"I'm driving"**
3. Enter name, phone number, origin, destination, date, departure time, flexibility
4. Ride posted and visible on the board
5. Receive push notification if a searcher requests that route

### Searcher Flow
1. Land on homepage
2. Tap **"I need a ride"**
3. Browse existing rides by origin/destination
4. If match found → view driver's name and phone number → contact directly
5. If no match → post a waiting request with same fields
6. Receive push notification when a matching ride is posted
