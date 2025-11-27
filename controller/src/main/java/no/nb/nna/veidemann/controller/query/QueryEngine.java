/*
 * Copyright 2018 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package no.nb.nna.veidemann.controller.query;

import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.util.Objects;

import com.google.gson.Gson;
import com.google.gson.JsonSyntaxException;
import com.google.gson.annotations.SerializedName;
import com.rethinkdb.ast.ReqlAst;

public class QueryEngine {
    private final HttpClient httpClient;
    private final URI endpoint;
    private final Gson gson;
    
    public QueryEngine(URI endpoint) {
        this(HttpClient.newHttpClient(), endpoint);
    }

    public QueryEngine(HttpClient httpClient, URI endpoint) {
        this.httpClient = Objects.requireNonNull(httpClient);
        this.endpoint = Objects.requireNonNull(endpoint);
        this.gson = new Gson();
    }

    public ReqlAst parseQuery(String query) throws IOException {
        String jsonBody = gson.toJson(new QueryRequest(query));

        HttpRequest request = HttpRequest.newBuilder(endpoint)
                .header("Content-Type", "application/json")
                .POST(HttpRequest.BodyPublishers.ofString(jsonBody, StandardCharsets.UTF_8))
                .build();

        HttpResponse<String> response;
        try {
            response = httpClient.send(
                    request,
                    HttpResponse.BodyHandlers.ofString(StandardCharsets.UTF_8));
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new IOException("Interrupted while calling AST service", e);
        }

        if (response.statusCode() != 200) {
            throw new IOException(
                    "AST service returned " + response.statusCode() + ": " + response.body());
        }

        QueryResponse parsed;
        try {
            parsed = gson.fromJson(response.body(), QueryResponse.class);
        } catch (JsonSyntaxException e) {
            throw new IllegalStateException("Malformed JSON from AST service", e);
        }

        if (parsed == null || parsed.ast == null) {
            throw new IllegalStateException("Missing 'ast' field in AST service response: " + response.body());
        }

        return new RethinkPreparsedTerm(parsed.ast);
    }

    // Request DTO
    private static class QueryRequest {
        final String query;
        QueryRequest(String query) {
            this.query = query;
        }
    }

    // Response DTO
    private static class QueryResponse {
        @SerializedName("ast")
        String ast;
    }
}
