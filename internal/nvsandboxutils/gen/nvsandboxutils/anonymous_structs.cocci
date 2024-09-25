@patch@
type WRAPPER_TYPE;
field list FIELDS;
identifier V;
expression E;
fresh identifier ST = "nvSandboxUtilsGenerated_struct___";
fresh identifier TEMP_VAR = "nvSandboxUtilsGenerated_variable___" ## V;
@@

++ struct ST {
++    WRAPPER_TYPE TEMP_VAR;
++    FIELDS
++ };
+

WRAPPER_TYPE
{
    ...
(
-    struct {
-       FIELDS
-   } V[E];
+   struct ST V[E];

|

-    struct {
-       FIELDS
-   } V;
+   struct ST V;
)
    ...
};

@capture@
type WRAPPER_TYPE;
identifier TEMP_VAR;
identifier ST =~ "^nvSandboxUtilsGenerated_struct___";
@@

struct ST {
  WRAPPER_TYPE TEMP_VAR;
  ...
};

@script:python concat@
WRAPPER_TYPE << capture.WRAPPER_TYPE;
TEMP_VAR << capture.TEMP_VAR;
ST << capture.ST;
T;
@@

def removePrefix(string, prefix):
    if string.startswith(prefix):
        return string[len(prefix):]
    return string

def removeSuffix(string, suffix):
    if string.endswith(suffix):
        return string[:-len(suffix)]
    return string

WRAPPER_TYPE = removeSuffix(WRAPPER_TYPE, "_t")
TEMP_VAR = removePrefix(TEMP_VAR, "nvSandboxUtilsGenerated_variable___")
coccinelle.T = cocci.make_type(WRAPPER_TYPE + TEMP_VAR[0].upper() + TEMP_VAR[1:] + "_t")

@add_typedef@
identifier capture.ST;
type concat.T;
type WRAPPER_TYPE;
identifier TEMP_VAR;
@@

- struct ST {
+ typedef struct {
- WRAPPER_TYPE TEMP_VAR;
  ...
- };
+ } T;

@update@
identifier capture.ST;
type concat.T;
identifier V;
expression E;
type WRAPPER_TYPE;
@@

WRAPPER_TYPE
{
    ...
(
-   struct ST V[E];
+   T V[E];
|
-   struct ST V;
+   T V;
)
    ...
};
