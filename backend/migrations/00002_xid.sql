-- +goose Up

-- The domain is named xid20, not xid, because pg_catalog.xid (the 4-byte
-- transaction-id type) is always resolved before the search_path. A column
-- declared as an unqualified `xid` would silently become a transaction id in
-- any schema, so the domain carries a name that cannot be shadowed.

-- +goose StatementBegin
do
$$
begin
    create domain public.xid20 as char(20) check (value ~ '^[a-v0-9]{20}$');
exception
    when duplicate_object then null;
end
$$;
-- +goose StatementEnd

create sequence if not exists public.xid_serial minvalue 0 maxvalue 16777215 cycle;
select setval('public.xid_serial', (random() * 16777215)::int);

-- +goose StatementBegin
create or replace function public.xid_encode(_id int[]) returns public.xid20 language plpgsql as
$$
declare
    _encoding char(1)[] = '{0,1,2,3,4,5,6,7,8,9,a,b,c,d,e,f,g,h,i,j,k,l,m,n,o,p,q,r,s,t,u,v}';
begin
    return _encoding[1+(_id[1]>>3)]||_encoding[1+((_id[2]>>6)&31|(_id[1]<<2)&31)]||_encoding[1+((_id[2]>>1)&31)]||_encoding[1+((_id[3]>>4)&31|(_id[2]<<4)&31)]||_encoding[1+(_id[4]>>7|(_id[3]<<1)&31)]||_encoding[1+((_id[4]>>2)&31)]||_encoding[1+(_id[5]>>5|(_id[4]<<3)&31)]||_encoding[1+(_id[5]&31)]||_encoding[1+(_id[6]>>3)]||_encoding[1+((_id[7]>>6)&31|(_id[6]<<2)&31)]||_encoding[1+((_id[7]>>1)&31)]||_encoding[1+((_id[8]>>4)&31|(_id[7]<<4)&31)]||_encoding[1+(_id[9]>>7|(_id[8]<<1)&31)]||_encoding[1+((_id[9]>>2)&31)]||_encoding[1+(_id[10]>>5|(_id[9]<<3)&31)]||_encoding[1+(_id[10]&31)]||_encoding[1+(_id[11]>>3)]||_encoding[1+((_id[12]>>6)&31|(_id[11]<<2)&31)]||_encoding[1+((_id[12]>>1)&31)]||_encoding[1+((_id[12]<<4)&31)];
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function public.xid(_at timestamptz default current_timestamp) returns public.xid20 language plpgsql as
$$
declare _t int; _m int; _p int; _c int;
begin
    _t:=floor(extract(epoch from _at)); _m:=(select (system_identifier&16777215)::int from pg_control_system()); _p:=pg_backend_pid(); _c:=nextval('public.xid_serial')::int;
    return public.xid_encode(array[(_t>>24)&255,(_t>>16)&255,(_t>>8)&255,_t&255,(_m>>16)&255,(_m>>8)&255,_m&255,(_p>>8)&255,_p&255,(_c>>16)&255,(_c>>8)&255,_c&255]);
end;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function public.xid_decode(_xid public.xid20) returns int[] language plpgsql immutable as
$$
declare
    _alphabet text := '0123456789abcdefghijklmnopqrstuv';
    _value int;
    _bits int := 0;
    _byte int := 0;
    _result int[] := '{}';
    _character text;
begin
    foreach _character in array string_to_array(lower(_xid::text), null) loop
        _value := strpos(_alphabet, _character) - 1;
        _byte := (_byte << 5) | _value;
        _bits := _bits + 5;
        while _bits >= 8 loop
            _bits := _bits - 8;
            _result := array_append(_result, (_byte >> _bits) & 255);
        end loop;
    end loop;
    return _result;
end;
$$;
-- +goose StatementEnd

-- The component accessors below parenthesize every shift because PostgreSQL
-- gives + higher precedence than << and gives << and | equal precedence.

-- +goose StatementBegin
create or replace function public.xid_time(_xid public.xid20) returns timestamptz language sql immutable as
$$ select to_timestamp((_b[1]::bigint<<24)+(_b[2]::bigint<<16)+(_b[3]::bigint<<8)+_b[4]) from (select public.xid_decode(_xid) as _b) _d $$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function public.xid_machine(_xid public.xid20) returns int[] language sql immutable as
$$ select array[_b[5],_b[6],_b[7]] from (select public.xid_decode(_xid) as _b) _d $$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function public.xid_pid(_xid public.xid20) returns int language sql immutable as
$$ select (_b[8]<<8)|_b[9] from (select public.xid_decode(_xid) as _b) _d $$;
-- +goose StatementEnd

-- +goose StatementBegin
create or replace function public.xid_counter(_xid public.xid20) returns int language sql immutable as
$$ select (_b[10]<<16)|(_b[11]<<8)|_b[12] from (select public.xid_decode(_xid) as _b) _d $$;
-- +goose StatementEnd

alter table health_checks
    alter column id drop default,
    alter column id type public.xid20 using public.xid(),
    alter column id set default public.xid();

-- +goose Down
alter table health_checks
    alter column id drop default,
    alter column id type uuid using gen_random_uuid(),
    alter column id set default uuidv7();

drop function if exists public.xid_counter(public.xid20);
drop function if exists public.xid_pid(public.xid20);
drop function if exists public.xid_machine(public.xid20);
drop function if exists public.xid_time(public.xid20);
drop function if exists public.xid_decode(public.xid20);
drop function if exists public.xid(timestamptz);
drop function if exists public.xid_encode(int[]);
drop sequence if exists public.xid_serial;
drop domain if exists public.xid20;
