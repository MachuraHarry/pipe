package gen

type builtinInfo struct {
	arity    int
	argTypes []PipeType
}

func (bi builtinInfo) argType(idx int) PipeType {
	if idx < len(bi.argTypes) {
		return bi.argTypes[idx]
	}
	return TypeAny
}

var builtinInfos = map[string]builtinInfo{
	"print":          {1, strs(TypeAny)},
	"read_file":      {1, strs(TypeString)},
	"write_file":     {2, strs(TypeString, TypeString)},
	"file_exists":    {1, strs(TypeString)},
	"path_join":      {2, strs(TypeString, TypeString)},
	"path_base":      {1, strs(TypeString)},
	"path_ext":       {1, strs(TypeString)},
	"env":            {1, strs(TypeString)},
	"sleep":          {1, strs(TypeInt)},
	"upper":          {1, strs(TypeString)},
	"lower":          {1, strs(TypeString)},
	"trim":           {1, strs(TypeString)},
	"split":          {2, strs(TypeString, TypeString)},
	"join":           {2, strs(TypeString, TypeString)},
	"contains":       {2, strs(TypeString, TypeString)},
	"len":            {1, strs(TypeAny)},
	"push":           {2, strs(TypeAny, TypeAny)},
	"pop":            {1, strs(TypeAny)},
	"at":             {2, strs(TypeAny, TypeInt)},
	"sort":           {1, strs(TypeAny)},
	"range":          {2, strs(TypeInt, TypeInt)},
	"abs":            {1, strs(TypeInt)},
	"min":            {2, strs(TypeInt, TypeInt)},
	"max":            {2, strs(TypeInt, TypeInt)},
	"pow":            {2, strs(TypeInt, TypeInt)},
	"sqrt":           {1, strs(TypeInt)},
	"round":          {1, strs(TypeInt)},
	"keys":           {1, strs(TypeAny)},
	"values":         {1, strs(TypeAny)},
	"get":            {2, strs(TypeAny, TypeAny)},
	"set":            {3, strs(TypeAny, TypeAny, TypeAny)},
	"type_of":        {1, strs(TypeAny)},
	"is_num":         {1, strs(TypeAny)},
	"is_str":         {1, strs(TypeAny)},
	"is_list":        {1, strs(TypeAny)},
	"is_map":         {1, strs(TypeAny)},
	"is_nil":         {1, strs(TypeAny)},
	"to_str":         {1, strs(TypeAny)},
	"to_num":         {1, strs(TypeAny)},
	"Ok":             {1, strs(TypeAny)},
	"Err":            {1, strs(TypeAny)},
	"is_ok":          {1, strs(TypeAny)},
	"is_err":         {1, strs(TypeAny)},
	"unwrap":         {1, strs(TypeAny)},
	"unwrap_or":      {2, strs(TypeAny, TypeAny)},
	"base64_encode":  {1, strs(TypeAny)},
	"base64_decode":  {1, strs(TypeAny)},
	"regex_match":    {2, strs(TypeString, TypeString)},
	"regex_replace":  {3, strs(TypeString, TypeString, TypeString)},
	"now":            {0, nil},
	"format_time":    {2, strs(TypeInt, TypeString)},
	"random":         {0, nil},
	"random_range":   {2, strs(TypeInt, TypeInt)},
	"parse_json":     {1, strs(TypeString)},
	"to_json":        {1, strs(TypeAny)},
	"http_get":       {1, strs(TypeString)},
	"ask":            {1, strs(TypeString)},
	"summarize":      {1, strs(TypeString)},
	"translate":      {2, strs(TypeString, TypeString)},
	"classify":       {2, strs(TypeString, TypeString)},
	"extract":        {2, strs(TypeString, TypeString)},
	"generate":       {1, strs(TypeString)},
	"generate_json":  {2, strs(TypeString, TypeString)},
	"ai_provider":    {1, strs(TypeString)},
	"ai_model":       {1, strs(TypeString)},
	"ai_timeout":     {1, strs(TypeInt)},
	"ai_host":        {1, strs(TypeString)},
	"ai_cache":       {1, strs(TypeAny)},
	"web_search":     {1, strs(TypeString)},
	"agent":          {2, strs(TypeString, TypeString)},
	"agent_ask":      {2, strs(TypeString, TypeString)},
	"agent_clear":    {1, strs(TypeString)},
	"ai_chat":        {2, strs(TypeString, TypeAny)},
	"embed":          {1, strs(TypeString)},
	"cosine_sim":     {2, strs(TypeAny, TypeAny)},
	"nearest":        {2, strs(TypeAny, TypeAny)},
	"input":          {1, strs(TypeString)},
}

func strs(types ...PipeType) []PipeType {
	return types
}
